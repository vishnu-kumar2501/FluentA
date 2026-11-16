//go:build windows

package desktop

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procMoveWindow                    = user32.NewProc("MoveWindow")
	procSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	procGetGUIThreadInfo              = user32.NewProc("GetGUIThreadInfo")
	procGetWindowRect                 = user32.NewProc("GetWindowRect")
	procGetClassNameW                 = user32.NewProc("GetClassNameW")
	procGetWindowLongPtrW             = user32.NewProc("GetWindowLongPtrW")
	procSendMessageTimeoutW           = user32.NewProc("SendMessageTimeoutW")
	procUpdateWindow                  = user32.NewProc("UpdateWindow")
	procAttachThreadInput             = user32.NewProc("AttachThreadInput")
	procBringWindowToTop              = user32.NewProc("BringWindowToTop")
	procSetFocus                      = user32.NewProc("SetFocus")
	procEnableWindow                  = user32.NewProc("EnableWindow")
	procGetCursorPos                  = user32.NewProc("GetCursorPos")
	procMonitorFromPoint              = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	procRegisterHotKey                = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey              = user32.NewProc("UnregisterHotKey")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procIsDialogMessageW              = user32.NewProc("IsDialogMessageW")
	procPostMessageW                  = user32.NewProc("PostMessageW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procCallWindowProcW               = user32.NewProc("CallWindowProcW")
	procSetWindowLongPtrW             = user32.NewProc("SetWindowLongPtrW")
	procSendMessageW                  = user32.NewProc("SendMessageW")
	procSetWindowTextW                = user32.NewProc("SetWindowTextW")
	procGetWindowTextW                = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW          = user32.NewProc("GetWindowTextLengthW")
	procSendInput                     = user32.NewProc("SendInput")
	procGetAsyncKeyState              = user32.NewProc("GetAsyncKeyState")
	procOpenClipboard                 = user32.NewProc("OpenClipboard")
	procCloseClipboard                = user32.NewProc("CloseClipboard")
	procEmptyClipboard                = user32.NewProc("EmptyClipboard")
	procGetClipboardData              = user32.NewProc("GetClipboardData")
	procSetClipboardData              = user32.NewProc("SetClipboardData")
	procIsClipboardFormatAvailable    = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardSequenceNumber    = user32.NewProc("GetClipboardSequenceNumber")
	procLoadIconW                     = user32.NewProc("LoadIconW")
	procLoadCursorW                   = user32.NewProc("LoadCursorW")
	procCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	procAppendMenuW                   = user32.NewProc("AppendMenuW")
	procTrackPopupMenu                = user32.NewProc("TrackPopupMenu")
	procDestroyMenu                   = user32.NewProc("DestroyMenu")
	procMessageBoxW                   = user32.NewProc("MessageBoxW")
	procGetSysColorBrush              = user32.NewProc("GetSysColorBrush")
	procBeginPaint                    = user32.NewProc("BeginPaint")
	procEndPaint                      = user32.NewProc("EndPaint")
	procInvalidateRect                = user32.NewProc("InvalidateRect")
	procFillRect                      = user32.NewProc("FillRect")
	procDrawTextW                     = user32.NewProc("DrawTextW")
	procReleaseCapture                = user32.NewProc("ReleaseCapture")
	procSetCapture                    = user32.NewProc("SetCapture")
	procSetCursor                     = user32.NewProc("SetCursor")

	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId         = kernel32.NewProc("GetCurrentThreadId")
	procGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	procGlobalFree                 = kernel32.NewProc("GlobalFree")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	procLstrlenW                   = kernel32.NewProc("lstrlenW")
	procRtlMoveMemory              = kernel32.NewProc("RtlMoveMemory")
	procCreateMutexW               = kernel32.NewProc("CreateMutexW")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procCreateFontW        = gdi32.NewProc("CreateFontW")
	procCreateSolidBrush   = gdi32.NewProc("CreateSolidBrush")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procSetBkMode          = gdi32.NewProc("SetBkMode")
	procSetTextColor       = gdi32.NewProc("SetTextColor")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procShellNotifyIconW      = shell32.NewProc("Shell_NotifyIconW")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	wsOverlapped   = 0x00000000
	wsPopup        = 0x80000000
	wsChild        = 0x40000000
	wsVisible      = 0x10000000
	wsCaption      = 0x00C00000
	wsBorder       = 0x00800000
	wsSysMenu      = 0x00080000
	wsThickFrame   = 0x00040000
	wsMinimizeBox  = 0x00020000
	wsMaximizeBox  = 0x00010000
	wsTabStop      = 0x00010000
	wsVScroll      = 0x00200000
	wsClipChildren = 0x02000000
	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000

	bsPushButton    = 0x00000000
	bsDefPushButton = 0x00000001
	esMultiline     = 0x0004
	esAutoVScroll   = 0x0040
	esWantReturn    = 0x1000
	cbsDropDownList = 0x0003

	wmDestroy          = 0x0002
	wmMove             = 0x0003
	wmSize             = 0x0005
	wmSetFont          = 0x0030
	wmGetMinMaxInfo    = 0x0024
	wmWindowPosChanged = 0x0047
	wmNCCalcSize       = 0x0083
	wmNCLButtonDown    = 0x00A1
	wmClose            = 0x0010
	wmQueryEndSession  = 0x0011
	wmEndSession       = 0x0016
	wmCommand     = 0x0111
	wmKeyDown     = 0x0100
	wmChar        = 0x0102
	wmHotKey      = 0x0312
	wmPaint          = 0x000F
	wmSetCursor      = 0x0020
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmLButtonDbl     = 0x0203
	wmRButtonUp      = 0x0205
	wmCaptureChanged = 0x0215
	idcHand          = 32649
	wmApp            = 0x8000
	bnClicked     = 0
	cbnSelChange  = 1
	cbAddString   = 0x0143
	cbSetCurSel   = 0x014E
	cbGetCurSel   = 0x0147
	emSetSel      = 0x00B1
	emReplaceSel  = 0x00C2
	emScrollCaret = 0x00B7

	swHide           = 0
	swShowNoActivate = 4
	swShow           = 5
	hwndTopmost      = ^uintptr(0) // (HWND)-1
	swpNoSize        = 0x0001
	swpNoMove        = 0x0002
	swpNoZOrder      = 0x0004
	swpFrameChanged  = 0x0020
	swpNoActivate    = 0x0010
	swpShowWin       = 0x0040
	htCaption        = 2
	htLeft           = 10
	htRight          = 11
	htTop            = 12
	htBottom         = 15
	htBottomRight    = 17
	vkLButton        = 0x01
	dwmwaDarkMode    = 20
	dwmwaCornerPref  = 33
	dwmwaBorderColor = 34
	dwmwcpRound      = 2
	dtCenter         = 0x00000001
	dtVCenter        = 0x00000004
	dtSingleLine     = 0x00000020
	transparentBk    = 1
	colorWindow  = 5
	colorBtnFace = 15
	gwlpWndProc  = ^uintptr(3) // GWLP_WNDPROC (-4)
	gwlStyle     = ^uintptr(15) // GWL_STYLE (-16)

	wmGetText       = 0x000D
	wmGetTextLength = 0x000E
	wmGetObject     = 0x003D
	objidClient     = ^uintptr(3) // OBJID_CLIENT (-4)
	smtoAbortIfHung = 0x0002
	esPassword      = 0x0020
	esReadOnly      = 0x0800

	vkReturn   = 0x0D
	vkEscape   = 0x1B
	vkShift    = 0x10
	vkControl  = 0x11
	vkMenu     = 0x12
	vkLShift   = 0xA0
	vkRShift   = 0xA1
	vkLControl = 0xA2
	vkRControl = 0xA3
	vkLMenu    = 0xA4
	vkRMenu    = 0xA5
	vkLWin     = 0x5B
	vkRWin     = 0x5C

	inputKeyboard    = 1
	keyeventfKeyUp   = 0x0002
	cfUnicodeText    = 13
	gmemMoveable     = 0x0002
	monitorNearest   = 2
	idiApplication   = 32512
	idcArrow         = 32512
	nimAdd                     = 0x00000000
	nimModify                  = 0x00000001
	nimDelete                  = 0x00000002
	errHotkeyAlreadyRegistered = syscall.Errno(1409)
	nifMessage       = 0x00000001
	nifIcon          = 0x00000002
	nifTip           = 0x00000004
	mfString         = 0x00000000
	mfSeparator      = 0x00000800
	tpmReturnCmd     = 0x0100
	tpmRightAlign    = 0x0008
	tpmBottomAlign   = 0x0020
	mbIconError      = 0x00000010
	mbIconInfo       = 0x00000040
	dpiContextPerMon = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
)

type point struct {
	X int32
	Y int32
}

type minMaxInfo struct {
	PtReserved     point
	PtMaxSize      point
	PtMaxPosition  point
	PtMinTrackSize point
	PtMaxTrackSize point
}

type guiThreadInfo struct {
	CbSize        uint32
	Flags         uint32
	HwndActive    uintptr
	HwndFocus     uintptr
	HwndCapture   uintptr
	HwndMenuOwner uintptr
	HwndMoveSize  uintptr
	HwndCaret     uintptr
	RcCaret       Rect
}

type winMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type monitorInfo struct {
	CbSize    uint32
	RcMonitor Rect
	RcWork    Rect
	DwFlags   uint32
}

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	_           uint32
	dwExtraInfo uintptr
}

type winInput struct {
	inputType uint32
	_         uint32
	ki        keybdInput
	_         [8]byte
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

func enableDPIAwareness() {
	_, _, _ = procSetProcessDpiAwarenessContext.Call(dpiContextPerMon)
}

func getModuleHandle() uintptr {
	h, _, _ := procGetModuleHandleW.Call(0)
	return h
}

func getForegroundWindow() uintptr {
	h, _, _ := procGetForegroundWindow.Call()
	return h
}

func setWindowText(hwnd uintptr, text string) {
	_, _, _ = procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(utf16Ptr(text))))
}

func getWindowText(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, int(n)+1)
	_, _, _ = procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func sendMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procSendMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func postMessage(hwnd uintptr, msg uint32, wParam, lParam uintptr) {
	_, _, _ = procPostMessageW.Call(hwnd, uintptr(msg), wParam, lParam)
}

func createSegoeFont(height int32) uintptr {
	h, _, _ := procCreateFontW.Call(
		uintptr(height),
		0, 0, 0,
		400, // FW_NORMAL
		0, 0, 0,
		1, // DEFAULT_CHARSET
		0, 0, 0, 0,
		uintptr(unsafe.Pointer(utf16Ptr("Segoe UI"))),
	)
	return h
}

func getCursorWorkArea() (point, Rect) {
	var pt point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt, getWorkAreaForPoint(pt)
}

func getWorkAreaForPoint(pt point) Rect {
	// MonitorFromPoint takes POINT by value (packed X in low 32 bits, Y in high 32 bits on amd64).
	packedPt := uintptr(uint32(pt.X)) | (uintptr(uint32(pt.Y)) << 32)
	mon, _, _ := procMonitorFromPoint.Call(packedPt, monitorNearest)
	var mi monitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	if mon != 0 {
		r, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		if r != 0 {
			return mi.RcWork
		}
	}
	return Rect{Left: 0, Top: 0, Right: 1280, Bottom: 720}
}

type paintStruct struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     Rect
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

func focusWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	fg := getForegroundWindow()
	if fg == hwnd {
		return
	}
	fgThread, _, _ := procGetWindowThreadProcessId.Call(fg, 0)
	curThread, _, _ := procGetCurrentThreadId.Call()
	attached := false
	if fgThread != 0 && fgThread != curThread {
		r, _, _ := procAttachThreadInput.Call(curThread, fgThread, 1)
		attached = r != 0
	}
	_, _, _ = procBringWindowToTop.Call(hwnd)
	_, _, _ = procSetForegroundWindow.Call(hwnd)
	if attached {
		_, _, _ = procAttachThreadInput.Call(curThread, fgThread, 0)
	}
}

func sendInputs(inputs []winInput) error {
	if len(inputs) == 0 {
		return nil
	}
	sent, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(winInput{}),
	)
	if int(sent) != len(inputs) {
		return fmt.Errorf("SendInput sent %d/%d events: %w", sent, len(inputs), err)
	}
	return nil
}

func isKeyDown(vk uint16) bool {
	state, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return (uint16(state) & 0x8000) != 0
}

func releaseHeldModifiers() error {
	modifiers := []uint16{
		vkShift, vkLShift, vkRShift,
		vkMenu, vkLMenu, vkRMenu,
		vkLWin, vkRWin,
		vkControl, vkLControl, vkRControl,
	}
	var up []winInput
	for _, vk := range modifiers {
		if isKeyDown(vk) {
			up = append(up, winInput{
				inputType: inputKeyboard,
				ki: keybdInput{
					wVk:     vk,
					dwFlags: keyeventfKeyUp,
				},
			})
		}
	}
	return sendInputs(up)
}

func sendCtrlChord(vk uint16) error {
	if err := releaseHeldModifiers(); err != nil {
		return err
	}
	seq := []winInput{
		{inputType: inputKeyboard, ki: keybdInput{wVk: vkControl}},
		{inputType: inputKeyboard, ki: keybdInput{wVk: vk}},
		{inputType: inputKeyboard, ki: keybdInput{wVk: vk, dwFlags: keyeventfKeyUp}},
		{inputType: inputKeyboard, ki: keybdInput{wVk: vkControl, dwFlags: keyeventfKeyUp}},
	}
	return sendInputs(seq)
}

func showMessageBox(title, body string, flags uint32) {
	_, _, _ = procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr(body))),
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(flags),
	)
}

func getWindowClassName(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	var buf [128]uint16
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

func wakeWindowAccessibility(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	var result uintptr
	_, _, _ = procSendMessageTimeoutW.Call(
		hwnd,
		wmGetObject,
		0,
		objidClient,
		smtoAbortIfHung,
		50,
		uintptr(unsafe.Pointer(&result)),
	)
}

func getFocusedChildHWND(fgHWND uintptr) uintptr {
	if fgHWND == 0 {
		return 0
	}
	fgThread, _, _ := procGetWindowThreadProcessId.Call(fgHWND, 0)
	if fgThread == 0 {
		return 0
	}
	var info guiThreadInfo
	info.CbSize = uint32(unsafe.Sizeof(info))
	r, _, _ := procGetGUIThreadInfo.Call(fgThread, uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0
	}
	return info.HwndFocus
}

func readWin32EditText(hwnd uintptr) (string, Rect, bool) {
	if hwnd == 0 {
		return "", Rect{}, false
	}
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	if style&esPassword != 0 || style&esReadOnly != 0 {
		return "", Rect{}, false
	}

	var length uintptr
	r, _, _ := procSendMessageTimeoutW.Call(hwnd, wmGetTextLength, 0, 0, smtoAbortIfHung, 50, uintptr(unsafe.Pointer(&length)))
	if r == 0 || length == 0 {
		return "", Rect{}, false
	}
	if length > 4096 {
		length = 4096
	}
	buf := make([]uint16, int(length)+1)
	var copied uintptr
	r, _, _ = procSendMessageTimeoutW.Call(
		hwnd,
		wmGetText,
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&buf[0])),
		smtoAbortIfHung,
		50,
		uintptr(unsafe.Pointer(&copied)),
	)
	if r == 0 || copied == 0 {
		return "", Rect{}, false
	}

	var bounds Rect
	_, _, _ = procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds)))
	return syscall.UTF16ToString(buf[:copied]), bounds, true
}

func configureFramelessWindow(hwnd uintptr, dark bool) {
	if hwnd == 0 {
		return
	}
	var darkVal uint32
	var borderColor uint32 = 0x00CFD8DC // Light #DCD8CF in COLORREF (0x00BBGGRR)
	if dark {
		darkVal = 1
		borderColor = 0x0042352F // Dark #2F3542 in COLORREF (0x00BBGGRR)
	}
	cornerPref := uint32(dwmwcpRound)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, dwmwaDarkMode, uintptr(unsafe.Pointer(&darkVal)), unsafe.Sizeof(darkVal))
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, dwmwaCornerPref, uintptr(unsafe.Pointer(&cornerPref)), unsafe.Sizeof(cornerPref))
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, dwmwaBorderColor, uintptr(unsafe.Pointer(&borderColor)), unsafe.Sizeof(borderColor))
	_, _, _ = procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
}

func startWindowDrag(hwnd uintptr) {
	if hwnd == 0 || !isKeyDown(vkLButton) {
		return
	}
	_, _, _ = procReleaseCapture.Call()
	_, _, _ = procSendMessageW.Call(hwnd, wmNCLButtonDown, htCaption, 0)
}

func startWindowResize(hwnd uintptr, edge string) {
	if hwnd == 0 || !isKeyDown(vkLButton) {
		return
	}
	var ht uintptr
	switch edge {
	case "e":
		ht = htRight
	case "s":
		ht = htBottom
	case "se":
		ht = htBottomRight
	case "w":
		ht = htLeft
	case "n":
		ht = htTop
	default:
		return
	}
	_, _, _ = procReleaseCapture.Call()
	_, _, _ = procSendMessageW.Call(hwnd, wmNCLButtonDown, ht, 0)
}

const processQueryLimitedInformation = 0x1000

// getWindowProcessName resolves the executable basename (lowercase, e.g. "keepass.exe")
// owning hwnd, returning "" if unavailable or protected.
func getWindowProcessName(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	_, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	hProc, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if hProc == 0 {
		return ""
	}
	defer procCloseHandle.Call(hProc)

	buf := make([]uint16, 320)
	size := uint32(len(buf))
	r, _, _ := procQueryFullProcessImageNameW.Call(hProc, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 || size == 0 {
		return ""
	}
	full := syscall.UTF16ToString(buf[:size])
	return strings.ToLower(filepath.Base(full))
}
