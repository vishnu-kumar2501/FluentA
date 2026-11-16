//go:build windows

package desktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"grammar/ai"
	"grammar/grammar"
)

const (
	hotkeyID        = 1
	hotkeyProbeID   = 2
	trayIconID      = 1
	popupWidth      = 680
	popupHeight     = 580
	popupMinWidth   = 500
	popupMinHeight  = 420
	wmAppTray       = wmApp + 1
	wmAppChunk      = wmApp + 2
	wmAppDone       = wmApp + 3
	wmAppErr        = wmApp + 4
	wmAppExternalIn    = wmApp + 5
	wmAppExplain       = wmApp + 10
	wmAppAnalyze       = wmApp + 11
	wmAppChunkProgress = wmApp + 12

	trayMenuOpen   = 201
	trayMenuHealth = 202
	trayMenuExit   = 203
	trayMenuPause  = 205
)

// App manages the Windows system tray, global hotkey, background UIA watcher, clipboard capture, and WebView2 popup.
type App struct {
	checker     *grammar.Checker
	cfg         Config
	hotkeyMods       uint32
	hotkeyVK         uint32
	hotkeyLabel      string
	hotkeyPref       string
	hotkeyRegistered bool
	hotkeyNotice     string

	hwnd            uintptr
	badgeHWND       uintptr
	badgeFont       uintptr
	badgeDragging   bool
	badgeMouseDown  bool
	badgeDragStart  point
	badgeWinStart   point
	badgeUserMoved  bool
	badgeUserPos    point
	badgeLastTarget uintptr
	iconLarge       uintptr
	iconSmall      uintptr
	webview        *webViewHost
	trayAdded      bool
	targetHWND     uintptr
	backup         clipboardBackup
	hasBackup      bool
	replaceAllMode bool

	mu              sync.Mutex
	dataDir         string
	runCtx          context.Context
	watcherStarted  bool
	gen             uint64
	inferCancel     context.CancelFunc
	explainGen      uint64
	explainCancel   context.CancelFunc
	explainPending  *explainResult
	analyzeGen      uint64
	analyzeCancel   context.CancelFunc
	analyzePending  *analyzeResult
	currentMode     grammar.Mode
	defaultMode     grammar.Mode
	language        string
	theme           string
	excludedApps    []string
	history         []HistoryItem
	sourceText      string
	outputText      strings.Builder
	pendingChunk    strings.Builder
	pendingErr      string
	pendingProgress string
	watchEnabled    bool
	paused          bool
	popupVisible    bool
	badgePending    watchSnapshot
	badgeCurrent    watchSnapshot

	aiConnected    bool
	aiError        string
	aiProviderType string
	aiBaseURL      string
	aiModel        string
	aiApiKey       string
}

type explainResult struct {
	gen          uint64
	requestID    string
	explanations []grammar.ChangeExplanation
	errMsg       string
}

type analyzeResult struct {
	gen       uint64
	requestID string
	insights  *grammar.WritingInsights
	errMsg    string
}

var activeApp *App

// Run starts the Windows desktop FluentA app and blocks until exit or ctx cancellation.
func Run(ctx context.Context, checker *grammar.Checker, cfg Config) error {
	if checker == nil {
		return fmt.Errorf("%w: checker must not be nil", ai.ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Hotkey) == "" {
		cfg.Hotkey = "ctrl+alt+g"
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = grammar.ModeFix
	}

	dataDir := AppDataDir()
	closeLog, logErr := setupFileLogger(dataDir)
	if logErr != nil {
		log.Printf("file logger disabled (%v)", logErr)
	}
	defer closeLog()

	instLock, lockErr := acquireSingleInstance(instanceMutexName(dataDir))
	if lockErr != nil {
		if lockErr == ErrAlreadyRunning {
			showMessageBox("FluentA", "FluentA is already running in the system tray.", mbIconInfo)
		}
		return lockErr
	}
	defer instLock.Release()

	settings, hasSettings, settingsErr := LoadSettings(dataDir)
	if settingsErr != nil {
		log.Printf("settings load warning: %v", settingsErr)
	}
	historyItems, histErr := LoadHistory(dataDir)
	if histErr != nil {
		log.Printf("history load warning: %v", histErr)
	}

	watchEnabled := cfg.WatchEnabled
	defaultMode := cfg.DefaultMode
	paused := false
	language := settings.Language
	theme := settings.Theme
	excludedApps := settings.ExcludedApps
	if hasSettings {
		if settings.WatchEnabled != nil {
			watchEnabled = *settings.WatchEnabled
		}
		paused = settings.Paused
		if parsedMode, err := grammar.ParseMode(settings.DefaultMode); err == nil && parsedMode != "" {
			defaultMode = parsedMode
		}
		if strings.EqualFold(strings.TrimSpace(cfg.Hotkey), "ctrl+alt+g") && strings.TrimSpace(settings.Hotkey) != "" {
			if _, _, _, err := ParseHotkey(settings.Hotkey); err == nil {
				cfg.Hotkey = settings.Hotkey
			}
		}
	} else if len(excludedApps) == 0 {
		excludedApps = DefaultExcludedApps()
	}

	normSettings := NormalizeSettings(settings)
	aiProvType := normSettings.AIProviderType
	aiBaseURL := normSettings.AIBaseURL
	aiModel := normSettings.AIModel
	aiApiKey := normSettings.AIApiKey

	// Verify AI readiness without hard crash so user can onboard or adjust settings gracefully
	aiConnected := true
	aiError := ""
	checkCtx, checkCancel := context.WithTimeout(ctx, 4*time.Second)
	if err := checker.VerifyReady(checkCtx); err != nil {
		aiConnected = false
		aiError = err.Error()
		log.Printf("initial AI readiness notice: %v", err)
	}
	checkCancel()

	mods, vk, label, err := ParseHotkey(cfg.Hotkey)
	if err != nil {
		return err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	enableDPIAwareness()

	app := &App{
		checker:        checker,
		cfg:            cfg,
		hotkeyMods:     mods,
		hotkeyVK:       vk,
		hotkeyLabel:    label,
		hotkeyPref:     label,
		dataDir:        dataDir,
		runCtx:         ctx,
		currentMode:    defaultMode,
		defaultMode:    defaultMode,
		language:       language,
		theme:          theme,
		excludedApps:   excludedApps,
		history:        historyItems,
		watchEnabled:   watchEnabled,
		paused:         paused,
		aiConnected:    aiConnected,
		aiError:        aiError,
		aiProviderType: aiProvType,
		aiBaseURL:      aiBaseURL,
		aiModel:        aiModel,
		aiApiKey:       aiApiKey,
	}
	activeApp = app
	defer func() {
		activeApp = nil
	}()

	if err := app.initWindow(); err != nil {
		return err
	}
	defer app.cleanup()

	if !aiConnected {
		// Launch window visibly on startup so user is guided through AI onboarding
		app.showPopupNearCursor()
	}

	if watchEnabled && !paused {
		app.ensureWatcherStarted()
	}

	go func() {
		<-ctx.Done()
		if app.hwnd != 0 {
			postMessage(app.hwnd, wmAppExternalIn, 0, 0)
		}
	}()

	log.Printf("FluentA running in system tray. Hotkey: %s | Background UIA watcher: %v (model=%s, ui=webview2).",
		app.hotkeyLabel, cfg.WatchEnabled, cfg.ModelName)

	var m winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func (a *App) initWindow() error {
	hInst := getModuleHandle()
	className := utf16Ptr("FluentAPopupClass")
	a.iconLarge, a.iconSmall = loadFluentAIcons(hInst)
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	bgBrush, _, _ := procGetSysColorBrush.Call(colorWindow)

	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInst,
		HIcon:         a.iconLarge,
		HCursor:       cursor,
		HbrBackground: bgBrush,
		LpszClassName: className,
		HIconSm:       a.iconSmall,
	}
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW failed: %w", err)
	}

	title := fmt.Sprintf("FluentA (%s)", a.hotkeyLabel)
	hwnd, _, winErr := procCreateWindowExW.Call(
		wsExTopmost|wsExToolWindow,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		wsOverlapped|wsThickFrame|wsClipChildren,
		100, 100, popupWidth, popupHeight,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed: %w", winErr)
	}
	a.hwnd = hwnd
	_, _, _ = procSendMessageW.Call(a.hwnd, wmSetIcon, iconBigID, a.iconLarge)
	_, _, _ = procSendMessageW.Call(a.hwnd, wmSetIcon, iconSmallID, a.iconSmall)

	wv, err := newWebViewHost(a.hwnd, a.handleUIMessage)
	if err != nil {
		return err
	}
	a.webview = wv
	configureFramelessWindow(a.hwnd, true)

	if err := a.initBadgeWindow(hInst); err != nil {
		return err
	}
	a.mu.Lock()
	startPaused := a.paused
	a.mu.Unlock()
	if !startPaused {
		if err := a.registerAvailableHotkey(); err != nil {
			log.Printf("hotkey startup warning: %v", err)
		}
	}

	setWindowText(a.hwnd, a.windowTitle())
	a.sendFullConfig()
	a.addTrayIcon(a.iconSmall)
	return nil
}

func (a *App) windowTitle() string {
	if !a.paused && !a.hotkeyRegistered {
		return "FluentA (Hotkey conflict)"
	}
	return fmt.Sprintf("FluentA (%s)", a.hotkeyLabel)
}

func (a *App) registerHotkeyID(id uintptr, mods, vk uint32) (ok bool, conflict bool, err error) {
	r, _, hkErr := procRegisterHotKey.Call(a.hwnd, id, uintptr(mods), uintptr(vk))
	if r != 0 {
		return true, false, nil
	}
	isConflict := errors.Is(hkErr, errHotkeyAlreadyRegistered)
	return false, isConflict, hkErr
}

// probeHotkey validates spec and performs a non-destructive Windows RegisterHotKey check
// using a separate probe ID so the active shortcut stays registered.
func (a *App) probeHotkey(spec string) (mods uint32, vk uint32, label string, available bool, conflict bool, isCurrent bool, errMsg string) {
	mods, vk, label, err := ParseHotkey(spec)
	if err != nil {
		return 0, 0, "", false, false, false, err.Error()
	}

	a.mu.Lock()
	activeAlready := !a.paused && a.hotkeyRegistered && mods == a.hotkeyMods && vk == a.hotkeyVK
	a.mu.Unlock()
	if activeAlready {
		return mods, vk, label, true, false, true, ""
	}
	if a.hwnd == 0 {
		return mods, vk, label, true, false, false, ""
	}

	ok, isConflict, hkErr := a.registerHotkeyID(hotkeyProbeID, mods, vk)
	if ok {
		_, _, _ = procUnregisterHotKey.Call(a.hwnd, hotkeyProbeID)
		return mods, vk, label, true, false, false, ""
	}
	if isConflict {
		return mods, vk, label, false, true, false, fmt.Sprintf("%s is already registered by another app or Windows", label)
	}
	return mods, vk, label, false, false, false, fmt.Sprintf("Could not test %s: %v", label, hkErr)
}

func (a *App) registerAvailableHotkey() error {
	preferred := strings.TrimSpace(a.hotkeyPref)
	if preferred == "" {
		preferred = a.cfg.Hotkey
	}
	_, _, prefLabel, prefErr := ParseHotkey(preferred)
	if prefErr != nil {
		prefLabel = preferred
	}

	candidates := []string{preferred}
	for _, fb := range []string{"ctrl+alt+g", "ctrl+shift+alt+g", "ctrl+alt+f8", "ctrl+alt+f9"} {
		if !strings.EqualFold(preferred, fb) {
			candidates = append(candidates, fb)
		}
	}

	var lastErr error
	var hadConflict bool
	for i, spec := range candidates {
		mods, vk, label, err := ParseHotkey(spec)
		if err != nil {
			lastErr = err
			continue
		}
		ok, isConflict, hkErr := a.registerHotkeyID(hotkeyID, mods, vk)
		if ok {
			a.mu.Lock()
			a.hotkeyMods = mods
			a.hotkeyVK = vk
			a.hotkeyLabel = label
			a.hotkeyRegistered = true
			if i == 0 {
				a.hotkeyNotice = ""
			} else {
				a.hotkeyNotice = fmt.Sprintf("Shortcut %s is already used by another app; using %s instead. Customize it in Preferences.", prefLabel, label)
			}
			notice := a.hotkeyNotice
			a.mu.Unlock()
			if notice != "" {
				log.Print(notice)
			}
			return nil
		}
		if isConflict {
			hadConflict = true
		}
		lastErr = hkErr
	}

	a.mu.Lock()
	a.hotkeyRegistered = false
	if hadConflict {
		a.hotkeyNotice = fmt.Sprintf("Shortcut %s is already used by another app. Open Preferences to choose a free global shortcut.", prefLabel)
	} else {
		a.hotkeyNotice = fmt.Sprintf("Could not register %s (%v). Open Preferences to choose another shortcut.", prefLabel, lastErr)
	}
	a.mu.Unlock()
	return fmt.Errorf("RegisterHotKey (%s) failed: %w", prefLabel, lastErr)
}

func (a *App) switchHotkey(spec string) error {
	mods, vk, label, avail, isConflict, isCurrent, errMsg := a.probeHotkey(spec)
	if !avail {
		if isConflict {
			return fmt.Errorf("%s is already used by another app", label)
		}
		return errors.New(errMsg)
	}

	a.mu.Lock()
	oldMods := a.hotkeyMods
	oldVK := a.hotkeyVK
	wasReg := a.hotkeyRegistered
	isPaused := a.paused
	a.mu.Unlock()

	if !isPaused && !isCurrent && a.hwnd != 0 {
		if wasReg {
			_, _, _ = procUnregisterHotKey.Call(a.hwnd, hotkeyID)
		}
		ok, conflict, hkErr := a.registerHotkeyID(hotkeyID, mods, vk)
		if !ok {
			if wasReg {
				_, _, _ = a.registerHotkeyID(hotkeyID, oldMods, oldVK)
			}
			if conflict {
				return fmt.Errorf("%s is already used by another app", label)
			}
			return fmt.Errorf("failed to register %s: %w", label, hkErr)
		}
	}

	a.mu.Lock()
	a.hotkeyMods = mods
	a.hotkeyVK = vk
	a.hotkeyLabel = label
	a.hotkeyPref = label
	a.hotkeyNotice = ""
	if !isPaused {
		a.hotkeyRegistered = true
	}
	a.mu.Unlock()

	if a.hwnd != 0 {
		setWindowText(a.hwnd, a.windowTitle())
		a.updateTrayTooltip()
	}
	return nil
}

func (a *App) trayTooltipText() string {
	return a.windowTitle()
}

func (a *App) addTrayIcon(icon uintptr) {
	var nid notifyIconData
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = trayIconID
	nid.UFlags = nifMessage | nifIcon | nifTip
	nid.UCallbackMessage = wmAppTray
	nid.HIcon = icon
	tip, _ := syscall.UTF16FromString(a.trayTooltipText())
	copy(nid.SzTip[:], tip)

	r, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	a.trayAdded = r != 0
}

func (a *App) updateTrayTooltip() {
	if !a.trayAdded || a.hwnd == 0 {
		return
	}
	var nid notifyIconData
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = a.hwnd
	nid.UID = trayIconID
	nid.UFlags = nifTip
	tip, _ := syscall.UTF16FromString(a.trayTooltipText())
	copy(nid.SzTip[:], tip)
	_, _, _ = procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func (a *App) cleanup() {
	a.cancelInference()
	if a.badgeHWND != 0 {
		_, _, _ = procDestroyWindow.Call(a.badgeHWND)
		a.badgeHWND = 0
	}
	if a.badgeFont != 0 {
		_, _, _ = procDeleteObject.Call(a.badgeFont)
		a.badgeFont = 0
	}
	if a.hwnd != 0 {
		_, _, _ = procUnregisterHotKey.Call(a.hwnd, hotkeyID)
		if a.trayAdded {
			var nid notifyIconData
			nid.CbSize = uint32(unsafe.Sizeof(nid))
			nid.HWnd = a.hwnd
			nid.UID = trayIconID
			_, _, _ = procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
			a.trayAdded = false
		}
		_, _, _ = procDestroyWindow.Call(a.hwnd)
		a.hwnd = 0
	}
	if a.iconSmall != 0 && a.iconSmall != a.iconLarge {
		destroyIcon(a.iconSmall)
	}
	destroyIcon(a.iconLarge)
	a.iconSmall = 0
	a.iconLarge = 0
}

func (a *App) activeMode() grammar.Mode {
	a.mu.Lock()
	m := a.currentMode
	a.mu.Unlock()
	if m == "" {
		return grammar.ModeFix
	}
	return m
}

func (a *App) setMode(mode grammar.Mode) {
	a.mu.Lock()
	a.currentMode = mode
	a.mu.Unlock()
}

func (a *App) onHotkey() {
	a.cancelInference()
	a.targetHWND = getForegroundWindow()
	a.replaceAllMode = false

	captured, backup, ok, err := captureSelection()
	a.backup = backup
	a.hasBackup = err == nil

	a.showPopupNearCursor()
	if err != nil {
		a.sourceText = ""
		a.webview.send(hostEvent{
			Type:             "open",
			Mode:             string(a.activeMode()),
			CanReplaceTarget: a.targetHWND != 0,
			HotkeyLabel:      a.hotkeyLabel,
			ModelName:        a.cfg.ModelName,
			Error:            "Clipboard capture error: " + err.Error(),
		})
		return
	}

	if !ok || strings.TrimSpace(captured) == "" {
		a.sourceText = ""
		a.webview.send(hostEvent{
			Type:             "open",
			Mode:             string(a.activeMode()),
			CanReplaceTarget: a.targetHWND != 0,
			HotkeyLabel:      a.hotkeyLabel,
			ModelName:        a.cfg.ModelName,
		})
		return
	}

	a.sourceText = captured
	a.webview.send(hostEvent{
		Type:             "open",
		SourceText:       captured,
		Mode:             string(a.activeMode()),
		CanReplaceTarget: a.targetHWND != 0,
		HotkeyLabel:      a.hotkeyLabel,
		ModelName:        a.cfg.ModelName,
		Streaming:        true,
	})
	a.startInference(captured)
}

func (a *App) openBlankPopup() {
	a.cancelInference()
	a.targetHWND = 0
	a.hasBackup = false
	a.replaceAllMode = false

	a.showPopupNearCursor()
	existing := strings.TrimSpace(a.sourceText)
	if existing != "" {
		a.webview.send(hostEvent{
			Type:        "open",
			SourceText:  a.sourceText,
			Mode:        string(a.activeMode()),
			HotkeyLabel: a.hotkeyLabel,
			ModelName:   a.cfg.ModelName,
			Streaming:   true,
		})
		a.startInference(a.sourceText)
		return
	}

	a.webview.send(hostEvent{
		Type:        "open",
		Mode:        string(a.activeMode()),
		HotkeyLabel: a.hotkeyLabel,
		ModelName:   a.cfg.ModelName,
	})
}

func (a *App) openFromWatcher(snap watchSnapshot) {
	a.cancelInference()
	a.targetHWND = snap.targetHWND
	a.hasBackup = false
	a.replaceAllMode = true
	a.sourceText = snap.sourceText

	a.showPopupNearCursor()

	switch snap.state {
	case badgeReady:
		a.setMode(grammar.ModeFix)
		a.mu.Lock()
		a.outputText.Reset()
		a.outputText.WriteString(snap.suggestion)
		a.mu.Unlock()
		a.webview.send(hostEvent{
			Type:             "open",
			SourceText:       snap.sourceText,
			OutputText:       snap.suggestion,
			Mode:             string(grammar.ModeFix),
			ReplaceAllMode:   true,
			CanReplaceTarget: true,
			HotkeyLabel:      a.hotkeyLabel,
			ModelName:        a.cfg.ModelName,
		})
	case badgeError:
		a.webview.send(hostEvent{
			Type:             "open",
			SourceText:       snap.sourceText,
			Mode:             string(a.activeMode()),
			ReplaceAllMode:   true,
			CanReplaceTarget: true,
			HotkeyLabel:      a.hotkeyLabel,
			ModelName:        a.cfg.ModelName,
			Error:            snap.errMsg,
		})
	default:
		a.webview.send(hostEvent{
			Type:             "open",
			SourceText:       snap.sourceText,
			Mode:             string(a.activeMode()),
			ReplaceAllMode:   true,
			CanReplaceTarget: true,
			HotkeyLabel:      a.hotkeyLabel,
			ModelName:        a.cfg.ModelName,
			Streaming:        true,
		})
		a.startInference(snap.sourceText)
	}
}

func (a *App) showPopupNearCursor() {
	a.mu.Lock()
	a.popupVisible = true
	a.mu.Unlock()
	if a.badgeHWND != 0 {
		_, _, _ = procShowWindow.Call(a.badgeHWND, swHide)
	}

	pt, work := getCursorWorkArea()
	x, y := ClampPopupPosition(pt.X, pt.Y, popupWidth, popupHeight, work)
	_, _, _ = procMoveWindow.Call(a.hwnd, uintptr(x), uintptr(y), popupWidth, popupHeight, 1)
	_, _, _ = procShowWindow.Call(a.hwnd, swShow)
	_, _, _ = procSetWindowPos.Call(a.hwnd, hwndTopmost, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpShowWin)
	a.webview.show()
	a.webview.resize()
	a.webview.notifyMoved()
	focusWindow(a.hwnd)
	a.webview.focus()
}

func (a *App) hidePopup() {
	a.cancelInference()
	a.mu.Lock()
	a.popupVisible = false
	a.mu.Unlock()
	a.webview.hide()
	_, _, _ = procShowWindow.Call(a.hwnd, swHide)
}

func (a *App) startInference(text string) {
	a.sourceText = text
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		a.webview.send(hostEvent{
			Type:  "error",
			Error: "Input text is empty.",
		})
		return
	}

	a.mu.Lock()
	connected := a.aiConnected
	aiErr := a.aiError
	a.mu.Unlock()
	if !connected {
		detail := "AI provider is not connected."
		if aiErr != "" {
			detail = "AI connection error: " + aiErr
		}
		a.webview.send(hostEvent{
			Type:  "error",
			Error: detail + " Open Preferences to configure your AI connection.",
		})
		return
	}

	mode := a.activeMode()
	lang := a.activeLanguage()
	a.cancelInference()

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.gen++
	gen := a.gen
	a.inferCancel = cancel
	a.outputText.Reset()
	a.pendingChunk.Reset()
	a.pendingErr = ""
	a.pendingProgress = ""
	a.mu.Unlock()

	a.webview.send(hostEvent{
		Type: "start",
		Mode: string(mode),
	})

	go func() {
		opts := grammar.CheckOptions{
			Mode:     mode,
			Language: lang,
			OnChunkProgress: func(idx, total int) {
				if total <= 1 {
					return
				}
				a.mu.Lock()
				isCurrent := a.gen == gen
				if isCurrent {
					a.pendingProgress = fmt.Sprintf("Chunk %d/%d...", idx, total)
				}
				a.mu.Unlock()
				if isCurrent {
					postMessage(a.hwnd, wmAppChunkProgress, uintptr(gen), 0)
				}
			},
		}
		_, err := a.checker.CheckWithOptionsStream(ctx, trimmed, opts, func(chunk ai.StreamChunk) error {
			if chunk.Content == "" {
				return nil
			}
			a.mu.Lock()
			if a.gen != gen {
				a.mu.Unlock()
				return context.Canceled
			}
			a.pendingChunk.WriteString(chunk.Content)
			a.outputText.WriteString(chunk.Content)
			a.mu.Unlock()
			postMessage(a.hwnd, wmAppChunk, uintptr(gen), 0)
			return nil
		})

		a.mu.Lock()
		isCurrent := a.gen == gen
		if isCurrent && err != nil {
			a.pendingErr = err.Error()
		}
		a.mu.Unlock()

		if !isCurrent {
			return
		}
		if err != nil {
			postMessage(a.hwnd, wmAppErr, uintptr(gen), 0)
			return
		}
		postMessage(a.hwnd, wmAppDone, uintptr(gen), 0)
	}()
}

func (a *App) cancelInference() {
	a.mu.Lock()
	cancel := a.inferCancel
	a.inferCancel = nil
	a.gen++
	a.pendingChunk.Reset()
	a.pendingErr = ""
	a.pendingProgress = ""
	exCancel := a.explainCancel
	a.explainCancel = nil
	a.explainGen++
	a.explainPending = nil
	anCancel := a.analyzeCancel
	a.analyzeCancel = nil
	a.analyzeGen++
	a.analyzePending = nil
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if exCancel != nil {
		exCancel()
	}
	if anCancel != nil {
		anCancel()
	}
}

func (a *App) flushPendingChunk(gen uint64) {
	a.mu.Lock()
	if a.gen != gen || a.pendingChunk.Len() == 0 {
		a.mu.Unlock()
		return
	}
	chunk := a.pendingChunk.String()
	a.pendingChunk.Reset()
	a.mu.Unlock()

	a.webview.send(hostEvent{
		Type: "chunk",
		Text: chunk,
	})
}

func (a *App) flushPendingProgress(gen uint64) {
	a.mu.Lock()
	if a.gen != gen || a.pendingProgress == "" {
		a.mu.Unlock()
		return
	}
	info := a.pendingProgress
	a.pendingProgress = ""
	a.mu.Unlock()

	a.webview.send(hostEvent{
		Type:      "chunkProgress",
		ChunkInfo: info,
	})
}

func (a *App) onInferenceDone(gen uint64) {
	a.flushPendingChunk(gen)
	a.mu.Lock()
	isCurrent := a.gen == gen
	finalOut := grammar.CleanOutput(a.outputText.String())
	srcText := a.sourceText
	mode := a.currentMode
	lang := a.language
	dataDir := a.dataDir
	a.mu.Unlock()
	if !isCurrent {
		return
	}
	a.webview.send(hostEvent{
		Type: "done",
		Text: finalOut,
	})
	if strings.TrimSpace(srcText) != "" && strings.TrimSpace(finalOut) != "" {
		updated, err := AppendHistory(dataDir, HistoryItem{
			Mode:       string(mode),
			Language:   lang,
			SourceText: srcText,
			OutputText: finalOut,
		}, defaultHistoryMaxItems)
		if err != nil {
			log.Printf("history save warning: %v", err)
			return
		}
		a.mu.Lock()
		a.history = updated
		a.mu.Unlock()
		a.webview.send(hostEvent{
			Type:    "history",
			History: updated,
		})
	}
}

func (a *App) onInferenceErr(gen uint64) {
	a.mu.Lock()
	if a.gen != gen {
		a.mu.Unlock()
		return
	}
	errMsg := a.pendingErr
	a.pendingErr = ""
	a.mu.Unlock()

	a.webview.send(hostEvent{
		Type:  "error",
		Error: errMsg,
	})
}

func (a *App) handleUIMessage(msg uiMessage) {
	switch msg.Type {
	case "ready":
		a.sendFullConfig()
	case "mode":
		if parsed, err := grammar.ParseMode(msg.Mode); err == nil {
			a.setMode(parsed)
		}
	case "recheck":
		if msg.Mode != "" {
			if parsed, err := grammar.ParseMode(msg.Mode); err == nil {
				a.setMode(parsed)
			}
		}
		a.startInference(msg.Text)
	case "stop":
		a.cancelInference()
	case "replace":
		a.onReplace(msg.Text)
	case "copy":
		a.onCopy(msg.Text)
	case "cancel":
		a.onCancel()
	case "drag":
		startWindowDrag(a.hwnd)
	case "resize":
		startWindowResize(a.hwnd, msg.Edge)
	case "theme":
		configureFramelessWindow(a.hwnd, msg.Theme == "dark")
	case "explainChanges":
		a.startExplainChanges(msg.RequestID, msg.Original, msg.Revised, msg.Changes)
	case "analyzeWriting":
		a.startAnalyzeWriting(msg.RequestID, msg.Text, msg.Language)
	case "setWatch":
		a.setWatchEnabled(msg.Enabled)
	case "setPause":
		a.setPaused(msg.Enabled)
	case "setLanguage":
		a.setLanguage(msg.Language)
	case "checkHotkey":
		spec := msg.Hotkey
		_, _, label, avail, conflict, _, errMsg := a.probeHotkey(spec)
		if a.webview != nil {
			a.webview.send(hostEvent{
				Type:            "hotkeyCheck",
				HotkeyLabel:     label,
				HotkeyAvailable: avail,
				HotkeyConflict:  conflict,
				Error:           errMsg,
			})
		}
	case "testAIConnection":
		go func() {
			provType := msg.AIProviderType
			baseURL := msg.AIBaseURL
			model := msg.AIModel
			apiKey := msg.AIApiKey
			a.mu.Lock()
			if provType == "" {
				provType = a.aiProviderType
			}
			if baseURL == "" {
				baseURL = a.aiBaseURL
			}
			if model == "" {
				model = a.aiModel
			}
			if apiKey == "" {
				apiKey = a.aiApiKey
			}
			a.mu.Unlock()

			ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
			defer cancel()
			err := TestProviderConnection(ctx, provType, baseURL, model, apiKey)
			var errMsg string
			connected := true
			if err != nil {
				connected = false
				errMsg = err.Error()
			}
			if a.webview != nil {
				a.webview.send(hostEvent{
					Type:        "aiTestResult",
					AIConnected: connected,
					AIError:     errMsg,
				})
			}
		}()
	case "saveSettings":
		if msg.Settings != nil {
			a.applySettings(*msg.Settings)
		}
	case "clearHistory":
		a.clearHistory()
	}
}

func (a *App) activeLanguage() string {
	a.mu.Lock()
	lang := a.language
	a.mu.Unlock()
	if strings.TrimSpace(lang) == "" {
		return "Auto-detect"
	}
	return lang
}

func (a *App) excludedAppsSnapshot() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, len(a.excludedApps))
	copy(out, a.excludedApps)
	return out
}

func (a *App) sendFullConfig() {
	if a.webview == nil {
		return
	}
	a.mu.Lock()
	watchOn := a.watchEnabled
	isPaused := a.paused
	hotkeyLbl := a.hotkeyLabel
	hotkeyNotice := a.hotkeyNotice
	hotkeyConflict := !a.paused && !a.hotkeyRegistered
	defMode := a.defaultMode
	lang := a.language
	theme := a.theme
	excl := make([]string, len(a.excludedApps))
	copy(excl, a.excludedApps)
	hist := make([]HistoryItem, len(a.history))
	copy(hist, a.history)
	a.mu.Unlock()

	if defMode == "" {
		defMode = grammar.ModeFix
	}
	if lang == "" {
		lang = "Auto-detect"
	}
	if theme == "" {
		theme = "system"
	}
	a.webview.send(hostEvent{
		Type:           "config",
		Mode:           string(a.activeMode()),
		DefaultMode:    string(defMode),
		HotkeyLabel:    hotkeyLbl,
		HotkeyConflict: hotkeyConflict,
		HotkeyNotice:   hotkeyNotice,
		ModelName:      a.cfg.ModelName,
		WatchEnabled:   watchOn,
		Paused:         isPaused,
		Language:       lang,
		Theme:          theme,
		ExcludedApps:   excl,
		History:        hist,
		AIConnected:    a.aiConnected,
		AIError:        a.aiError,
		AIProviderType: a.aiProviderType,
		AIBaseURL:      a.aiBaseURL,
		AIModel:        a.aiModel,
		AIApiKey:       a.aiApiKey,
	})
}

func (a *App) persistSettingsLocked() Settings {
	watch := a.watchEnabled
	defMode := a.defaultMode
	if defMode == "" {
		defMode = grammar.ModeFix
	}
	return NormalizeSettings(Settings{
		WatchEnabled:   &watch,
		Paused:         a.paused,
		Hotkey:         a.hotkeyPref,
		DefaultMode:    string(defMode),
		Language:       a.language,
		Theme:          a.theme,
		ExcludedApps:   a.excludedApps,
		AIProviderType: a.aiProviderType,
		AIBaseURL:      a.aiBaseURL,
		AIModel:        a.aiModel,
		AIApiKey:       a.aiApiKey,
	})
}

func (a *App) setLanguage(lang string) {
	a.mu.Lock()
	if strings.TrimSpace(lang) == "" {
		a.language = "Auto-detect"
	} else {
		a.language = strings.TrimSpace(lang)
	}
	s := a.persistSettingsLocked()
	dataDir := a.dataDir
	a.mu.Unlock()
	if err := SaveSettings(dataDir, s); err != nil {
		log.Printf("settings save warning: %v", err)
	}
}

func (a *App) applySettings(in Settings) {
	norm := NormalizeSettings(in)
	if norm.Hotkey != "" {
		if err := a.switchHotkey(norm.Hotkey); err != nil {
			log.Printf("hotkey change warning (%q): %v", norm.Hotkey, err)
			if a.webview != nil {
				a.webview.send(hostEvent{
					Type:            "hotkeyCheck",
					HotkeyAvailable: false,
					HotkeyConflict:  true,
					Error:           err.Error(),
				})
			}
		}
	}

	aiChanged := (norm.AIProviderType != a.aiProviderType || norm.AIBaseURL != a.aiBaseURL || norm.AIModel != a.aiModel || norm.AIApiKey != a.aiApiKey)
	if aiChanged || !a.aiConnected {
		newProv, provErr := BuildProvider(norm)
		if provErr == nil {
			if err := a.checker.SetProvider(newProv); err == nil {
				a.cfg.ModelName = norm.AIModel
				checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := a.checker.VerifyReady(checkCtx); err == nil {
					a.aiConnected = true
					a.aiError = ""
				} else {
					a.aiConnected = false
					a.aiError = err.Error()
				}
				cancel()
			}
		} else {
			a.aiConnected = false
			a.aiError = provErr.Error()
		}
		a.mu.Lock()
		a.aiProviderType = norm.AIProviderType
		a.aiBaseURL = norm.AIBaseURL
		a.aiModel = norm.AIModel
		a.aiApiKey = norm.AIApiKey
		a.mu.Unlock()
	}

	var nextWatch bool
	a.mu.Lock()
	if norm.WatchEnabled != nil {
		nextWatch = *norm.WatchEnabled
	} else {
		nextWatch = a.watchEnabled
	}
	if parsedMode, err := grammar.ParseMode(norm.DefaultMode); err == nil && parsedMode != "" {
		a.defaultMode = parsedMode
		a.currentMode = parsedMode
	}
	a.language = norm.Language
	a.theme = norm.Theme
	a.excludedApps = norm.ExcludedApps
	dataDir := a.dataDir
	a.mu.Unlock()

	a.setWatchEnabled(nextWatch)
	s := func() Settings {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.persistSettingsLocked()
	}()
	if err := SaveSettings(dataDir, s); err != nil {
		log.Printf("settings save warning: %v", err)
	}
	a.sendFullConfig()
}

func (a *App) clearHistory() {
	a.mu.Lock()
	a.history = nil
	dataDir := a.dataDir
	a.mu.Unlock()
	if err := ClearHistory(dataDir); err != nil {
		log.Printf("clear history warning: %v", err)
		return
	}
	a.webview.send(hostEvent{
		Type:    "history",
		History: []HistoryItem{},
	})
}

func (a *App) ensureWatcherStarted() {
	a.mu.Lock()
	alreadyStarted := a.watcherStarted
	ctx := a.runCtx
	if !alreadyStarted && ctx != nil {
		a.watcherStarted = true
	}
	a.mu.Unlock()
	if !alreadyStarted && ctx != nil {
		a.startWatcher(ctx)
	}
}

func (a *App) syncStatusToWebView() {
	if a.webview == nil {
		return
	}
	a.mu.Lock()
	watchOn := a.watchEnabled
	isPaused := a.paused
	hotkeyLbl := a.hotkeyLabel
	hotkeyNotice := a.hotkeyNotice
	hotkeyConflict := !a.paused && !a.hotkeyRegistered
	a.mu.Unlock()
	a.webview.send(hostEvent{
		Type:           "statusSync",
		HotkeyLabel:    hotkeyLbl,
		HotkeyConflict: hotkeyConflict,
		HotkeyNotice:   hotkeyNotice,
		WatchEnabled:   watchOn,
		Paused:         isPaused,
	})
}

func (a *App) setWatchEnabled(enabled bool) {
	a.mu.Lock()
	a.watchEnabled = enabled
	paused := a.paused
	s := a.persistSettingsLocked()
	dataDir := a.dataDir
	a.mu.Unlock()
	if enabled && !paused {
		a.ensureWatcherStarted()
	}
	if !enabled || paused {
		a.postBadgeState(watchSnapshot{state: badgeHidden})
	}
	if err := SaveSettings(dataDir, s); err != nil {
		log.Printf("settings save warning: %v", err)
	}
	a.syncStatusToWebView()
}

func (a *App) setPaused(paused bool) {
	a.mu.Lock()
	wasPaused := a.paused
	a.paused = paused
	watchOn := a.watchEnabled
	hwnd := a.hwnd
	s := a.persistSettingsLocked()
	dataDir := a.dataDir
	a.mu.Unlock()

	if hwnd != 0 && wasPaused != paused {
		if paused {
			_, _, _ = procUnregisterHotKey.Call(hwnd, hotkeyID)
			a.mu.Lock()
			a.hotkeyRegistered = false
			a.mu.Unlock()
		} else {
			if err := a.registerAvailableHotkey(); err != nil {
				log.Printf("hotkey re-register on resume failed: %v", err)
			}
			setWindowText(hwnd, a.windowTitle())
			a.updateTrayTooltip()
		}
	}
	if paused {
		a.postBadgeState(watchSnapshot{state: badgeHidden})
	} else if watchOn {
		a.ensureWatcherStarted()
	}
	if err := SaveSettings(dataDir, s); err != nil {
		log.Printf("settings save warning: %v", err)
	}
	a.syncStatusToWebView()
}

func (a *App) startAnalyzeWriting(reqID, text, language string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	a.mu.Lock()
	if a.analyzeCancel != nil {
		a.analyzeCancel()
	}
	a.analyzeGen++
	anGen := a.analyzeGen
	ctx, cancel := context.WithCancel(context.Background())
	a.analyzeCancel = cancel
	a.analyzePending = nil
	a.mu.Unlock()

	go func() {
		defer cancel()
		insights, err := a.checker.AnalyzeWriting(ctx, text, language)
		res := &analyzeResult{
			gen:       anGen,
			requestID: reqID,
		}
		if err != nil {
			res.errMsg = err.Error()
		} else {
			res.insights = &insights
		}

		a.mu.Lock()
		isCurrent := a.analyzeGen == anGen
		if isCurrent {
			a.analyzePending = res
		}
		a.mu.Unlock()
		if isCurrent {
			postMessage(a.hwnd, wmAppAnalyze, uintptr(anGen), 0)
		}
	}()
}

func (a *App) onAnalyzeReady(anGen uint64) {
	a.mu.Lock()
	if a.analyzeGen != anGen || a.analyzePending == nil {
		a.mu.Unlock()
		return
	}
	res := a.analyzePending
	a.analyzePending = nil
	a.mu.Unlock()

	if res.errMsg != "" {
		a.webview.send(hostEvent{
			Type:      "insights",
			RequestID: res.requestID,
			Error:     res.errMsg,
		})
		return
	}
	a.webview.send(hostEvent{
		Type:      "insights",
		RequestID: res.requestID,
		Insights:  res.insights,
	})
}

func (a *App) startExplainChanges(reqID, original, revised string, changes []grammar.DiffChange) {
	if len(changes) == 0 {
		return
	}
	a.mu.Lock()
	if a.explainCancel != nil {
		a.explainCancel()
	}
	a.explainGen++
	exGen := a.explainGen
	ctx, cancel := context.WithCancel(context.Background())
	a.explainCancel = cancel
	a.explainPending = nil
	a.mu.Unlock()

	go func() {
		defer cancel()
		explanations, err := a.checker.ExplainChanges(ctx, original, revised, changes)
		res := &explainResult{
			gen:          exGen,
			requestID:    reqID,
			explanations: explanations,
		}
		if err != nil {
			res.errMsg = err.Error()
		}

		a.mu.Lock()
		isCurrent := a.explainGen == exGen
		if isCurrent {
			a.explainPending = res
		}
		a.mu.Unlock()
		if isCurrent {
			postMessage(a.hwnd, wmAppExplain, uintptr(exGen), 0)
		}
	}()
}

func (a *App) onExplainReady(exGen uint64) {
	a.mu.Lock()
	if a.explainGen != exGen || a.explainPending == nil {
		a.mu.Unlock()
		return
	}
	res := a.explainPending
	a.explainPending = nil
	a.mu.Unlock()

	if res.errMsg != "" {
		a.webview.send(hostEvent{
			Type:      "explanations",
			RequestID: res.requestID,
			Error:     res.errMsg,
		})
		return
	}
	a.webview.send(hostEvent{
		Type:         "explanations",
		RequestID:    res.requestID,
		Explanations: res.explanations,
	})
}

func (a *App) onReplace(out string) {
	if out == "" {
		a.mu.Lock()
		out = a.outputText.String()
		a.mu.Unlock()
	}
	if strings.TrimSpace(out) == "" {
		a.webview.send(hostEvent{
			Type:  "error",
			Error: "Nothing to replace yet.",
		})
		return
	}

	target := a.targetHWND
	backup := a.backup
	hasBackup := a.hasBackup
	replaceAll := a.replaceAllMode

	a.hidePopup()
	if target == 0 {
		_ = writeClipboardText(out)
		return
	}
	if replaceAll {
		if err := replaceFieldAll(target, out); err != nil {
			log.Printf("replace all failed: %v", err)
		}
		a.hasBackup = false
		return
	}
	if !hasBackup {
		backup = clipboardBackup{}
	}
	if err := replaceSelection(target, out, backup); err != nil {
		log.Printf("replace failed: %v", err)
	}
	a.hasBackup = false
}

func (a *App) onCopy(out string) {
	if out == "" {
		a.mu.Lock()
		out = a.outputText.String()
		a.mu.Unlock()
	}
	if strings.TrimSpace(out) == "" {
		a.webview.send(hostEvent{
			Type:  "error",
			Error: "Nothing to copy yet.",
		})
		return
	}
	if err := writeClipboardText(out); err != nil {
		a.webview.send(hostEvent{
			Type:  "error",
			Error: "Copy failed: " + err.Error(),
		})
		return
	}
	a.hasBackup = false
}

func (a *App) onCancel() {
	a.hidePopup()
	if a.hasBackup {
		restoreClipboard(a.backup)
		a.hasBackup = false
	}
	if a.targetHWND != 0 {
		focusWindow(a.targetHWND)
	}
}

func (a *App) showTrayMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	a.mu.Lock()
	watching := a.watchEnabled
	isPaused := a.paused
	a.mu.Unlock()
	watchLabel := "Background Watcher: OFF (Enable)"
	if watching {
		watchLabel = "Background Watcher: ON (Turn Off)"
	}
	pauseLabel := "Pause Assistant (Hotkey & Watcher)"
	if isPaused {
		pauseLabel = "Resume Assistant (Currently Paused)"
	}

	_, _, _ = procAppendMenuW.Call(menu, mfString, trayMenuOpen, uintptr(unsafe.Pointer(utf16Ptr("Open FluentA"))))
	_, _, _ = procAppendMenuW.Call(menu, mfString, trayMenuToggleWatch, uintptr(unsafe.Pointer(utf16Ptr(watchLabel))))
	_, _, _ = procAppendMenuW.Call(menu, mfString, trayMenuPause, uintptr(unsafe.Pointer(utf16Ptr(pauseLabel))))
	_, _, _ = procAppendMenuW.Call(menu, mfString, trayMenuHealth, uintptr(unsafe.Pointer(utf16Ptr("Check Ollama Health"))))
	_, _, _ = procAppendMenuW.Call(menu, mfSeparator, 0, 0)
	_, _, _ = procAppendMenuW.Call(menu, mfString, trayMenuExit, uintptr(unsafe.Pointer(utf16Ptr("Exit"))))

	var pt point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	_, _, _ = procSetForegroundWindow.Call(a.hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightAlign|tpmBottomAlign, uintptr(pt.X), uintptr(pt.Y), 0, a.hwnd, 0)

	switch cmd {
	case trayMenuOpen:
		a.openBlankPopup()
	case trayMenuToggleWatch:
		a.setWatchEnabled(!watching)
	case trayMenuPause:
		a.setPaused(!isPaused)
	case trayMenuHealth:
		if err := a.checker.VerifyReady(context.Background()); err != nil {
			showMessageBox("FluentA - Health Check", "Ollama unavailable: "+err.Error(), mbIconError)
		} else {
			showMessageBox("FluentA - Health Check", fmt.Sprintf("Ollama is running and model %q is ready.", a.cfg.ModelName), mbIconInfo)
		}
	case trayMenuExit:
		_, _, _ = procPostQuitMessage.Call(0)
	}
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	app := activeApp
	if app == nil {
		r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	}

	switch msg {
	case wmHotKey:
		if wParam == hotkeyID {
			app.onHotkey()
		}
		return 0

	case wmSize:
		app.webview.resize()
		return 0

	case wmNCCalcSize:
		if wParam != 0 {
			return 0
		}

	case wmMove, wmWindowPosChanged:
		app.webview.notifyMoved()

	case wmGetMinMaxInfo:
		if lParam != 0 {
			var mmi minMaxInfo
			_, _, _ = procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&mmi)), lParam, unsafe.Sizeof(mmi))
			mmi.PtMinTrackSize.X = popupMinWidth
			mmi.PtMinTrackSize.Y = popupMinHeight
			_, _, _ = procRtlMoveMemory.Call(lParam, uintptr(unsafe.Pointer(&mmi)), unsafe.Sizeof(mmi))
			return 0
		}

	case wmAppTray:
		switch lParam {
		case wmRButtonUp:
			app.showTrayMenu()
		case wmLButtonDbl:
			app.openBlankPopup()
		}
		return 0

	case wmAppChunk:
		app.flushPendingChunk(uint64(wParam))
		return 0

	case wmAppDone:
		app.onInferenceDone(uint64(wParam))
		return 0

	case wmAppErr:
		app.onInferenceErr(uint64(wParam))
		return 0

	case wmAppExternalIn:
		_, _, _ = procPostQuitMessage.Call(0)
		return 0

	case wmAppBadgeUpdate:
		app.applyBadgeUpdate()
		return 0

	case wmAppExplain:
		app.onExplainReady(uint64(wParam))
		return 0

	case wmAppAnalyze:
		app.onAnalyzeReady(uint64(wParam))
		return 0

	case wmAppChunkProgress:
		app.flushPendingProgress(uint64(wParam))
		return 0

	case wmQueryEndSession:
		return 1

	case wmEndSession:
		if wParam != 0 {
			app.cleanup()
			_, _, _ = procPostQuitMessage.Call(0)
		}
		return 0

	case wmClose:
		app.onCancel()
		return 0

	case wmDestroy:
		_, _, _ = procPostQuitMessage.Call(0)
		return 0
	}

	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}
