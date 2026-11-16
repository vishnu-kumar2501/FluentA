//go:build windows

package desktop

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var (
	ole32    = syscall.NewLazyDLL("ole32.dll")
	oleaut32 = syscall.NewLazyDLL("oleaut32.dll")

	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")

	procSysFreeString = oleaut32.NewProc("SysFreeString")
	procSysStringLen  = oleaut32.NewProc("SysStringLen")
)

const (
	coinitMultithreaded = 0x0
	clsctxInprocServer  = 0x1

	uiaValuePatternID = 10002
	uiaTextPatternID  = 10014

	uiaComboBoxControlTypeID = 50003
	uiaEditControlTypeID     = 50004
	uiaTextControlTypeID     = 50020
	uiaCustomControlTypeID   = 50025
	uiaGroupControlTypeID    = 50026
	uiaDocumentControlTypeID = 50030

	maxWatchTextChars = 4096
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	clsidCUIAutomation = guid{
		Data1: 0xFF48DBA4, Data2: 0x60EF, Data3: 0x4201,
		Data4: [8]byte{0xAA, 0x87, 0x54, 0x10, 0x3E, 0xEF, 0x59, 0x4E},
	}
	iidIUIAutomation = guid{
		Data1: 0x30CBE57D, Data2: 0xD9D0, Data3: 0x452A,
		Data4: [8]byte{0xAB, 0x13, 0x7A, 0xC5, 0xAC, 0x48, 0x25, 0xEE},
	}
	iidIUIAutomationValuePattern = guid{
		Data1: 0xA94CD8B1, Data2: 0x0844, Data3: 0x4CD6,
		Data4: [8]byte{0x9D, 0x2D, 0x64, 0x05, 0x37, 0xAB, 0x39, 0xE9},
	}
	iidIUIAutomationTextPattern = guid{
		Data1: 0x32EBA289, Data2: 0x3583, Data3: 0x42C9,
		Data4: [8]byte{0x9C, 0x59, 0x3B, 0x6D, 0x9A, 0x1E, 0x9B, 0x6A},
	}
)

// FocusedField holds text and screen geometry captured from the focused Windows UI element.
type FocusedField struct {
	Text       string
	Bounds     Rect
	TargetHWND uintptr
}

// UIAReader reads focused editable controls via Windows UI Automation (COM) and Win32 fallbacks.
// Instances must be created and used on the same OS-locked MTA thread.
type UIAReader struct {
	uia           uintptr
	walker        uintptr
	selfPID       int32
	lastWokenHWND uintptr
	Debug         bool
	LastDebug     string
}

// NewUIAReader initializes COM (MTA) on the current thread and creates IUIAutomation.
func NewUIAReader() (*UIAReader, error) {
	hr, _, _ := procCoInitializeEx.Call(0, coinitMultithreaded)
	if int32(hr) < 0 && uint32(hr) != 0x80010106 {
		return nil, fmt.Errorf("CoInitializeEx failed: 0x%08x", uint32(hr))
	}

	var uia uintptr
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidCUIAutomation)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIUIAutomation)),
		uintptr(unsafe.Pointer(&uia)),
	)
	if int32(hr) < 0 || uia == 0 {
		procCoUninitialize.Call()
		return nil, fmt.Errorf("CoCreateInstance(CUIAutomation) failed: 0x%08x", uint32(hr))
	}

	var walker uintptr
	_ = comCall(uia, 14, uintptr(unsafe.Pointer(&walker))) // get_ControlViewWalker

	return &UIAReader{
		uia:     uia,
		walker:  walker,
		selfPID: int32(os.Getpid()),
	}, nil
}

// Close releases UIA COM objects and uninitializes COM on the current thread.
func (r *UIAReader) Close() {
	if r.walker != 0 {
		comRelease(r.walker)
		r.walker = 0
	}
	if r.uia != 0 {
		comRelease(r.uia)
		r.uia = 0
	}
	_, _, _ = procCoUninitialize.Call()
}

// ReadFocusedText inspects the currently focused UI element if it is an editable text field.
func (r *UIAReader) ReadFocusedText() (FocusedField, bool, error) {
	if r.uia == 0 {
		return FocusedField{}, false, fmt.Errorf("UIAReader is closed")
	}

	fgHWND := getForegroundWindow()
	if fgHWND != 0 && fgHWND != r.lastWokenHWND {
		// Signal Chromium/Electron/WinUI windows to enable their UIA accessibility tree.
		wakeWindowAccessibility(fgHWND)
		r.lastWokenHWND = fgHWND
	}

	var elem uintptr
	hr := comCall(r.uia, 8, uintptr(unsafe.Pointer(&elem))) // GetFocusedElement
	if int32(hr) >= 0 && elem != 0 {
		defer comRelease(elem)
		if field, ok := r.tryElementAndParents(elem, fgHWND); ok {
			return field, true, nil
		}
	} else if r.Debug {
		r.LastDebug = fmt.Sprintf("GetFocusedElement hr=0x%08x fg=%x", uint32(hr), fgHWND)
	}

	// Fallback: inspect Win32 focused child HWND via GetGUIThreadInfo.
	if childHWND := getFocusedChildHWND(fgHWND); childHWND != 0 {
		var childElem uintptr
		hr := comCall(r.uia, 6, childHWND, uintptr(unsafe.Pointer(&childElem))) // ElementFromHandle
		if int32(hr) >= 0 && childElem != 0 {
			defer comRelease(childElem)
			if field, ok := r.tryElement(childElem, fgHWND); ok {
				return field, true, nil
			}
		}

		if class := getWindowClassName(childHWND); isWin32EditClass(class) {
			if text, bounds, ok := readWin32EditText(childHWND); ok {
				if r.Debug {
					r.LastDebug = fmt.Sprintf("Win32 fallback hwnd=%x class=%q len=%d", childHWND, class, len(text))
				}
				return FocusedField{
					Text:       text,
					Bounds:     bounds,
					TargetHWND: fgHWND,
				}, true, nil
			}
		}
	}

	return FocusedField{}, false, nil
}

func (r *UIAReader) tryElementAndParents(elem, fgHWND uintptr) (FocusedField, bool) {
	if field, ok := r.tryElement(elem, fgHWND); ok {
		return field, true
	}
	if r.walker == 0 {
		return FocusedField{}, false
	}

	curr := elem
	needRelease := false
	for depth := 0; depth < 2; depth++ {
		var parent uintptr
		hr := comCall(r.walker, 3, curr, uintptr(unsafe.Pointer(&parent))) // GetParentElement
		if needRelease {
			comRelease(curr)
		}
		if int32(hr) < 0 || parent == 0 {
			return FocusedField{}, false
		}
		curr = parent
		needRelease = true
		if field, ok := r.tryElement(curr, fgHWND); ok {
			comRelease(curr)
			return field, true
		}
	}
	if needRelease {
		comRelease(curr)
	}
	return FocusedField{}, false
}

func (r *UIAReader) tryElement(elem, fgHWND uintptr) (FocusedField, bool) {
	var pid int32
	if hr := comCall(elem, 20, uintptr(unsafe.Pointer(&pid))); int32(hr) < 0 || pid == r.selfPID {
		return FocusedField{}, false
	}

	var isEnabled int32
	if hr := comCall(elem, 28, uintptr(unsafe.Pointer(&isEnabled))); int32(hr) < 0 || isEnabled == 0 {
		return FocusedField{}, false
	}

	var isPassword int32
	if hr := comCall(elem, 35, uintptr(unsafe.Pointer(&isPassword))); int32(hr) == 0 && isPassword != 0 {
		return FocusedField{}, false
	}

	var controlType int32
	if hr := comCall(elem, 21, uintptr(unsafe.Pointer(&controlType))); int32(hr) < 0 {
		return FocusedField{}, false
	}

	ariaRole := readElementStringProp(elem, 45)  // get_CurrentAriaRole
	className := readElementStringProp(elem, 30) // get_CurrentClassName

	var bounds Rect
	_ = comCall(elem, 43, uintptr(unsafe.Pointer(&bounds)))

	if isPotentialInputType(controlType, ariaRole) {
		if text, ok := readValuePattern(elem); ok {
			if r.Debug {
				r.LastDebug = fmt.Sprintf("UIA ValuePattern ct=%d role=%q class=%q len=%d", controlType, ariaRole, className, len(text))
			}
			return FocusedField{Text: text, Bounds: bounds, TargetHWND: fgHWND}, true
		}
	}

	if isEditableTextPatternTarget(controlType, ariaRole, className) {
		if text, ok := readTextPattern(elem); ok {
			if r.Debug {
				r.LastDebug = fmt.Sprintf("UIA TextPattern ct=%d role=%q class=%q len=%d", controlType, ariaRole, className, len(text))
			}
			return FocusedField{Text: text, Bounds: bounds, TargetHWND: fgHWND}, true
		}
	}

	if r.Debug {
		r.LastDebug = fmt.Sprintf("UIA skipped ct=%d role=%q class=%q", controlType, ariaRole, className)
	}
	return FocusedField{}, false
}

func isPotentialInputType(controlType int32, ariaRole string) bool {
	if strings.EqualFold(ariaRole, "textbox") {
		return true
	}
	switch controlType {
	case uiaEditControlTypeID, uiaDocumentControlTypeID, uiaComboBoxControlTypeID, uiaCustomControlTypeID, uiaGroupControlTypeID, uiaTextControlTypeID:
		return true
	default:
		return false
	}
}

func isEditableTextPatternTarget(controlType int32, ariaRole, className string) bool {
	if controlType == uiaEditControlTypeID || strings.EqualFold(ariaRole, "textbox") {
		return true
	}
	if controlType == uiaDocumentControlTypeID {
		lowerClass := strings.ToLower(className)
		if strings.Contains(lowerClass, "richedit") ||
			strings.Contains(lowerClass, "edit") ||
			strings.Contains(lowerClass, "_wwg") ||
			strings.Contains(lowerClass, "scintilla") {
			return true
		}
	}
	return false
}

func isWin32EditClass(className string) bool {
	lower := strings.ToLower(className)
	return lower == "edit" || strings.Contains(lower, "richedit")
}

func readElementStringProp(elem uintptr, slot int) string {
	var bstr uintptr
	if hr := comCall(elem, slot, uintptr(unsafe.Pointer(&bstr))); int32(hr) < 0 || bstr == 0 {
		return ""
	}
	defer procSysFreeString.Call(bstr)
	return bstrToString(bstr)
}

func readValuePattern(elem uintptr) (string, bool) {
	var unk uintptr
	hr := comCall(elem, 16, uiaValuePatternID, uintptr(unsafe.Pointer(&unk)))
	if int32(hr) < 0 || unk == 0 {
		return "", false
	}
	defer comRelease(unk)

	var pattern uintptr
	hr = comCall(unk, 0, uintptr(unsafe.Pointer(&iidIUIAutomationValuePattern)), uintptr(unsafe.Pointer(&pattern)))
	if int32(hr) < 0 || pattern == 0 {
		return "", false
	}
	defer comRelease(pattern)

	var isReadOnly int32
	if hr := comCall(pattern, 5, uintptr(unsafe.Pointer(&isReadOnly))); int32(hr) == 0 && isReadOnly != 0 {
		return "", false
	}

	var bstr uintptr
	if hr := comCall(pattern, 4, uintptr(unsafe.Pointer(&bstr))); int32(hr) < 0 || bstr == 0 {
		return "", false
	}
	defer procSysFreeString.Call(bstr)

	return bstrToString(bstr), true
}

func readTextPattern(elem uintptr) (string, bool) {
	var unk uintptr
	hr := comCall(elem, 16, uiaTextPatternID, uintptr(unsafe.Pointer(&unk)))
	if int32(hr) < 0 || unk == 0 {
		return "", false
	}
	defer comRelease(unk)

	var pattern uintptr
	hr = comCall(unk, 0, uintptr(unsafe.Pointer(&iidIUIAutomationTextPattern)), uintptr(unsafe.Pointer(&pattern)))
	if int32(hr) < 0 || pattern == 0 {
		return "", false
	}
	defer comRelease(pattern)

	var textRange uintptr
	if hr := comCall(pattern, 7, uintptr(unsafe.Pointer(&textRange))); int32(hr) < 0 || textRange == 0 {
		return "", false
	}
	defer comRelease(textRange)

	var bstr uintptr
	if hr := comCall(textRange, 12, uintptr(maxWatchTextChars), uintptr(unsafe.Pointer(&bstr))); int32(hr) < 0 || bstr == 0 {
		return "", false
	}
	defer procSysFreeString.Call(bstr)

	return bstrToString(bstr), true
}

func bstrToString(bstr uintptr) string {
	if bstr == 0 {
		return ""
	}
	n, _, _ := procSysStringLen.Call(bstr)
	length := int(n)
	if length <= 0 {
		return ""
	}
	if length > maxWatchTextChars {
		length = maxWatchTextChars
	}
	buf := make([]uint16, length)
	_, _, _ = procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&buf[0])), bstr, uintptr(length*2))
	return strings.TrimRight(syscall.UTF16ToString(buf), "\x00")
}

func comRelease(obj uintptr) {
	if obj != 0 {
		_ = comCall(obj, 2)
	}
}

func comCall(obj uintptr, slot int, args ...uintptr) uintptr {
	var vtbl uintptr
	_, _, _ = procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&vtbl)), obj, unsafe.Sizeof(vtbl))
	var fn uintptr
	_, _, _ = procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&fn)), vtbl+uintptr(slot)*unsafe.Sizeof(fn), unsafe.Sizeof(fn))

	allArgs := make([]uintptr, 0, 1+len(args))
	allArgs = append(allArgs, obj)
	allArgs = append(allArgs, args...)
	r, _, _ := syscall.SyscallN(fn, allArgs...)
	return r
}
