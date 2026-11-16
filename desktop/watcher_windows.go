//go:build windows

package desktop

import (
	"context"
	"errors"
	"log"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"grammar/grammar"
)

const (
	badgeWidth  int32 = 158
	badgeHeight int32 = 28

	wmAppBadgeUpdate = wmApp + 6

	trayMenuToggleWatch = 204
)

type badgeState int

const (
	badgeHidden badgeState = iota
	badgeChecking
	badgeClean
	badgeReady
	badgeError
)

type watchSnapshot struct {
	state      badgeState
	bounds     Rect
	sourceText string
	suggestion string
	errMsg     string
	targetHWND uintptr
}

func (a *App) initBadgeWindow(hInst uintptr) error {
	className := utf16Ptr("FluentAWatcherBadgeClass")
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)

	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   syscall.NewCallback(badgeWndProc),
		HInstance:     hInst,
		HCursor:       cursor,
		LpszClassName: className,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmtError("RegisterClassExW(badge)", err)
	}

	hwnd, _, winErr := procCreateWindowExW.Call(
		wsExTopmost|wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr("FluentA Status"))),
		wsPopup|wsBorder,
		0, 0, uintptr(badgeWidth), uintptr(badgeHeight),
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return fmtError("CreateWindowExW(badge)", winErr)
	}
	a.badgeHWND = hwnd
	a.badgeFont = createSegoeFont(-13)
	return nil
}

func (a *App) startWatcher(ctx context.Context) {
	go a.watcherLoop(ctx)
}

func (a *App) watcherLoop(ctx context.Context) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	reader, err := NewUIAReader()
	if err != nil {
		log.Printf("UI Automation watcher disabled (%v); Method 1 hotkey remains active.", err)
		return
	}
	reader.Debug = a.cfg.WatchDebug
	defer reader.Close()

	debounceDur := a.cfg.WatchDebounce
	if debounceDur <= 0 {
		debounceDur = 900 * time.Millisecond
	}

	const pollInterval = 350 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var (
		debounceTimer   *time.Timer
		debounceCh      <-chan time.Time
		pendingField    FocusedField
		lastSeenText    string
		lastCheckedText string
		lastBounds      Rect
		lastDebugLog    string
		missCount       int
		watchCancel     context.CancelFunc
	)

	stopDebounce := func() {
		if debounceTimer != nil {
			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			debounceTimer = nil
			debounceCh = nil
		}
	}

	cancelActiveCheck := func() {
		if watchCancel != nil {
			watchCancel()
			watchCancel = nil
		}
	}

	defer func() {
		stopDebounce()
		cancelActiveCheck()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-debounceCh:
			debounceTimer = nil
			debounceCh = nil

			if !a.isWatchActive() {
				continue
			}

			field := pendingField
			trimmed := strings.TrimSpace(toLF(field.Text))
			if !ShouldWatchText(trimmed) {
				continue
			}

			lastCheckedText = trimmed
			cancelActiveCheck()

			if reader.Debug {
				log.Printf("[watch-debug] checking %d chars with %s...", len(trimmed), a.cfg.ModelName)
			}

			checkCtx, cancel := context.WithCancel(ctx)
			watchCancel = cancel
			a.postBadgeState(watchSnapshot{
				state:      badgeChecking,
				bounds:     field.Bounds,
				sourceText: trimmed,
				targetHWND: field.TargetHWND,
			})

			go func(runCtx context.Context, runCancel context.CancelFunc, snap FocusedField, input string) {
				defer runCancel()
				resp, err := a.checker.CheckWithOptionsStream(runCtx, input, grammar.CheckOptions{
					Mode:     grammar.ModeFix,
					Language: a.activeLanguage(),
				}, nil)
				if runCtx.Err() != nil || errors.Is(err, context.Canceled) {
					return
				}
				if err != nil {
					log.Printf("watcher check failed: %v", err)
					a.postBadgeState(watchSnapshot{
						state:      badgeError,
						bounds:     snap.Bounds,
						sourceText: input,
						errMsg:     err.Error(),
						targetHWND: snap.TargetHWND,
					})
					return
				}

				corrected := strings.TrimSpace(toLF(resp.Message.Content))
				if corrected == "" || corrected == input {
					if reader.Debug {
						log.Printf("[watch-debug] result: grammar clean")
					}
					a.postBadgeState(watchSnapshot{
						state:      badgeClean,
						bounds:     snap.Bounds,
						sourceText: input,
						targetHWND: snap.TargetHWND,
					})
					return
				}

				if reader.Debug {
					log.Printf("[watch-debug] result: fix ready (%d chars)", len(corrected))
				}
				a.postBadgeState(watchSnapshot{
					state:      badgeReady,
					bounds:     snap.Bounds,
					sourceText: input,
					suggestion: corrected,
					targetHWND: snap.TargetHWND,
				})
			}(checkCtx, cancel, field, trimmed)

		case <-ticker.C:
			if !a.isWatchActive() {
				stopDebounce()
				cancelActiveCheck()
				lastSeenText = ""
				lastCheckedText = ""
				a.postBadgeState(watchSnapshot{state: badgeHidden})
				continue
			}

			field, ok, readErr := reader.ReadFocusedText()
			if reader.Debug && reader.LastDebug != "" && reader.LastDebug != lastDebugLog {
				lastDebugLog = reader.LastDebug
				log.Printf("[watch-debug] %s", lastDebugLog)
			}
			if readErr != nil || !ok {
				missCount++
				if missCount >= 2 {
					stopDebounce()
					cancelActiveCheck()
					lastSeenText = ""
					lastCheckedText = ""
					a.postBadgeState(watchSnapshot{state: badgeHidden})
				}
				continue
			}
			missCount = 0

			if procName := getWindowProcessName(field.TargetHWND); procName != "" && IsAppExcluded(procName, a.excludedAppsSnapshot()) {
				stopDebounce()
				cancelActiveCheck()
				lastSeenText = ""
				lastCheckedText = ""
				a.postBadgeState(watchSnapshot{state: badgeHidden})
				continue
			}

			trimmed := strings.TrimSpace(toLF(field.Text))
			if !ShouldWatchText(trimmed) {
				stopDebounce()
				cancelActiveCheck()
				lastSeenText = ""
				lastCheckedText = ""
				a.postBadgeState(watchSnapshot{state: badgeHidden})
				continue
			}

			if trimmed == lastSeenText {
				pendingField.Bounds = field.Bounds
				if trimmed == lastCheckedText && boundsMoved(lastBounds, field.Bounds) {
					lastBounds = field.Bounds
					a.updateBadgeBounds(field.Bounds)
				}
				continue
			}

			// Text changed: cancel stale inference and restart debounce timer.
			lastSeenText = trimmed
			lastBounds = field.Bounds
			pendingField = field
			cancelActiveCheck()
			stopDebounce()
			debounceTimer = time.NewTimer(debounceDur)
			debounceCh = debounceTimer.C
		}
	}
}

func (a *App) isWatchActive() bool {
	a.mu.Lock()
	enabled := a.watchEnabled && !a.paused && !a.popupVisible
	a.mu.Unlock()
	return enabled
}

func (a *App) postBadgeState(snap watchSnapshot) {
	a.mu.Lock()
	a.badgePending = snap
	hwnd := a.hwnd
	a.mu.Unlock()
	if hwnd != 0 {
		postMessage(hwnd, wmAppBadgeUpdate, 0, 0)
	}
}

func (a *App) updateBadgeBounds(bounds Rect) {
	a.mu.Lock()
	snap := a.badgeCurrent
	if snap.state == badgeHidden {
		a.mu.Unlock()
		return
	}
	snap.bounds = bounds
	a.badgePending = snap
	hwnd := a.hwnd
	a.mu.Unlock()
	if hwnd != 0 {
		postMessage(hwnd, wmAppBadgeUpdate, 0, 0)
	}
}

func (a *App) applyBadgeUpdate() {
	a.mu.Lock()
	snap := a.badgePending
	popupOpen := a.popupVisible
	enabled := a.watchEnabled && !a.paused
	if popupOpen || !enabled {
		snap.state = badgeHidden
	}
	a.badgeCurrent = snap
	isDragging := a.badgeDragging
	a.mu.Unlock()

	if a.badgeHWND == 0 {
		return
	}
	if snap.state == badgeHidden {
		a.badgeUserMoved = false
		_, _, _ = procShowWindow.Call(a.badgeHWND, swHide)
		return
	}

	if snap.targetHWND != a.badgeLastTarget {
		a.badgeLastTarget = snap.targetHWND
		a.badgeUserMoved = false
	}

	// While user is actively dragging the badge, do not interrupt with position updates.
	if isDragging {
		_, _, _ = procInvalidateRect.Call(a.badgeHWND, 0, 1)
		_, _, _ = procUpdateWindow.Call(a.badgeHWND)
		return
	}

	var x, y int32
	if a.badgeUserMoved {
		x = a.badgeUserPos.X
		y = a.badgeUserPos.Y
	} else {
		center := point{
			X: (snap.bounds.Left + snap.bounds.Right) / 2,
			Y: (snap.bounds.Top + snap.bounds.Bottom) / 2,
		}
		work := getWorkAreaForPoint(center)
		x, y = ClampBadgePosition(snap.bounds, badgeWidth, badgeHeight, work)
	}

	_, _, _ = procMoveWindow.Call(a.badgeHWND, uintptr(x), uintptr(y), uintptr(badgeWidth), uintptr(badgeHeight), 1)
	_, _, _ = procSetWindowPos.Call(a.badgeHWND, hwndTopmost, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoActivate|swpShowWin)
	_, _, _ = procShowWindow.Call(a.badgeHWND, swShowNoActivate)
	_, _, _ = procInvalidateRect.Call(a.badgeHWND, 0, 1)
	_, _, _ = procUpdateWindow.Call(a.badgeHWND)

	if snap.state == badgeClean {
		go func(expectedText string) {
			time.Sleep(2 * time.Second)
			a.mu.Lock()
			stillClean := a.badgeCurrent.state == badgeClean && a.badgeCurrent.sourceText == expectedText
			a.mu.Unlock()
			if stillClean {
				a.postBadgeState(watchSnapshot{state: badgeHidden})
			}
		}(snap.sourceText)
	}
}

func (a *App) onBadgeClicked() {
	a.mu.Lock()
	snap := a.badgeCurrent
	a.mu.Unlock()

	if snap.state == badgeHidden || strings.TrimSpace(snap.sourceText) == "" {
		return
	}
	a.openFromWatcher(snap)
}

func badgeWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	app := activeApp
	if app == nil {
		r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	}

	switch msg {
	case wmSetCursor:
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(idcHand))
		if cursor != 0 {
			_, _, _ = procSetCursor.Call(cursor)
			return 1
		}

	case wmLButtonDown:
		app.badgeMouseDown = true
		app.badgeDragging = false
		var pt point
		_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		app.badgeDragStart = pt

		var rc Rect
		_, _, _ = procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		app.badgeWinStart = point{X: rc.Left, Y: rc.Top}

		_, _, _ = procSetCapture.Call(hwnd)
		return 0

	case wmMouseMove:
		if app.badgeMouseDown {
			var pt point
			_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
			dx := pt.X - app.badgeDragStart.X
			dy := pt.Y - app.badgeDragStart.Y

			if !app.badgeDragging {
				if abs32(dx) > 3 || abs32(dy) > 3 {
					app.badgeDragging = true
				}
			}

			if app.badgeDragging {
				newX := app.badgeWinStart.X + dx
				newY := app.badgeWinStart.Y + dy
				work := getWorkAreaForPoint(point{X: newX, Y: newY})
				if newX < work.Left {
					newX = work.Left
				}
				if newX+badgeWidth > work.Right {
					newX = work.Right - badgeWidth
				}
				if newY < work.Top {
					newY = work.Top
				}
				if newY+badgeHeight > work.Bottom {
					newY = work.Bottom - badgeHeight
				}

				_, _, _ = procSetWindowPos.Call(hwnd, hwndTopmost, uintptr(newX), uintptr(newY), 0, 0, swpNoSize|swpNoActivate|swpShowWin)
				app.badgeUserMoved = true
				app.badgeUserPos = point{X: newX, Y: newY}
			}
		}
		return 0

	case wmLButtonUp:
		if app.badgeMouseDown {
			app.badgeMouseDown = false
			_, _, _ = procReleaseCapture.Call()

			if app.badgeDragging {
				app.badgeDragging = false
			} else {
				app.onBadgeClicked()
			}
		}
		return 0

	case wmCaptureChanged:
		app.badgeMouseDown = false
		app.badgeDragging = false
		return 0

	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			app.paintBadge(hdc)
			_, _, _ = procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		}
		return 0
	}

	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func (a *App) paintBadge(hdc uintptr) {
	a.mu.Lock()
	state := a.badgeCurrent.state
	a.mu.Unlock()

	var bgColor uintptr
	var label string
	switch state {
	case badgeChecking:
		bgColor = 0x00695547 // slate (#475569)
		label = "Checking..."
	case badgeClean:
		bgColor = 0x003D8015 // green (#15803D)
		label = "FluentA OK"
	case badgeReady:
		bgColor = 0x00D84E1D // blue (#1D4ED8)
		label = "Fix Ready • Click"
	case badgeError:
		bgColor = 0x001C1CB9 // red (#B91C1C)
		label = "Ollama Error"
	default:
		bgColor = 0x00695547
		label = "FluentA"
	}

	rc := Rect{Left: 0, Top: 0, Right: badgeWidth, Bottom: badgeHeight}
	brush, _, _ := procCreateSolidBrush.Call(bgColor)
	if brush != 0 {
		_, _, _ = procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brush)
		_, _, _ = procDeleteObject.Call(brush)
	}

	textRC := rc
	if a.iconSmall != 0 {
		drawIcon16(hdc, a.iconSmall, 6, (badgeHeight-16)/2)
		textRC.Left = 24
	}

	var oldFont uintptr
	if a.badgeFont != 0 {
		oldFont, _, _ = procSelectObject.Call(hdc, a.badgeFont)
	}
	_, _, _ = procSetBkMode.Call(hdc, transparentBk)
	_, _, _ = procSetTextColor.Call(hdc, 0x00FFFFFF)
	_, _, _ = procDrawTextW.Call(
		hdc,
		uintptr(unsafe.Pointer(utf16Ptr(label))),
		^uintptr(0), // -1 = null-terminated
		uintptr(unsafe.Pointer(&textRC)),
		dtCenter|dtVCenter|dtSingleLine,
	)
	if oldFont != 0 {
		_, _, _ = procSelectObject.Call(hdc, oldFont)
	}
}

func boundsMoved(a, b Rect) bool {
	return abs32(a.Left-b.Left) > 8 || abs32(a.Top-b.Top) > 8 ||
		abs32(a.Right-b.Right) > 8 || abs32(a.Bottom-b.Bottom) > 8
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func fmtError(op string, err error) error {
	if err != nil {
		return errors.New(op + " failed: " + err.Error())
	}
	return errors.New(op + " failed")
}
