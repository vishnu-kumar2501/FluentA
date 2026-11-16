//go:build windows

package desktop

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

type clipboardBackup struct {
	hadText bool
	text    string
}

func openClipboardWithRetry() error {
	for i := 0; i < 10; i++ {
		r, _, _ := procOpenClipboard.Call(0)
		if r != 0 {
			return nil
		}
		time.Sleep(15 * time.Millisecond)
	}
	return fmt.Errorf("failed to open Windows clipboard")
}

func readClipboardText() (string, bool, error) {
	avail, _, _ := procIsClipboardFormatAvailable.Call(cfUnicodeText)
	if avail == 0 {
		return "", false, nil
	}
	if err := openClipboardWithRetry(); err != nil {
		return "", false, err
	}
	defer procCloseClipboard.Call()

	hMem, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if hMem == 0 {
		return "", false, nil
	}
	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", false, fmt.Errorf("GlobalLock failed reading clipboard")
	}
	defer procGlobalUnlock.Call(hMem)

	return utf16PtrToString(ptr), true, nil
}

func writeClipboardText(text string) error {
	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("encode UTF-16 clipboard text: %w", err)
	}
	byteLen := uintptr(len(utf16) * 2)

	hMem, _, allocErr := procGlobalAlloc.Call(gmemMoveable, byteLen)
	if hMem == 0 {
		return fmt.Errorf("GlobalAlloc clipboard failed: %w", allocErr)
	}

	ptr, _, lockErr := procGlobalLock.Call(hMem)
	if ptr == 0 {
		_, _, _ = procGlobalFree.Call(hMem)
		return fmt.Errorf("GlobalLock clipboard failed: %w", lockErr)
	}

	_, _, _ = procRtlMoveMemory.Call(ptr, uintptr(unsafe.Pointer(&utf16[0])), byteLen)
	_, _, _ = procGlobalUnlock.Call(hMem)

	if err := openClipboardWithRetry(); err != nil {
		_, _, _ = procGlobalFree.Call(hMem)
		return err
	}
	defer procCloseClipboard.Call()

	_, _, _ = procEmptyClipboard.Call()
	r, _, setErr := procSetClipboardData.Call(cfUnicodeText, hMem)
	if r == 0 {
		_, _, _ = procGlobalFree.Call(hMem)
		return fmt.Errorf("SetClipboardData failed: %w", setErr)
	}
	return nil
}

func saveClipboard() (clipboardBackup, error) {
	text, ok, err := readClipboardText()
	if err != nil {
		return clipboardBackup{}, err
	}
	return clipboardBackup{hadText: ok, text: text}, nil
}

func restoreClipboard(backup clipboardBackup) {
	if !backup.hadText {
		return
	}
	_ = writeClipboardText(backup.text)
}

func captureSelection() (captured string, backup clipboardBackup, ok bool, err error) {
	backup, err = saveClipboard()
	if err != nil {
		return "", clipboardBackup{}, false, err
	}

	seqBefore, _, _ := procGetClipboardSequenceNumber.Call()

	// Brief delay helps ensure user's hotkey release settles before sending Ctrl+C.
	time.Sleep(40 * time.Millisecond)
	if err := sendCtrlChord('C'); err != nil {
		restoreClipboard(backup)
		return "", backup, false, err
	}

	deadline := time.Now().Add(350 * time.Millisecond)
	for time.Now().Before(deadline) {
		seqNow, _, _ := procGetClipboardSequenceNumber.Call()
		if seqNow != seqBefore {
			text, hasText, readErr := readClipboardText()
			if readErr != nil {
				restoreClipboard(backup)
				return "", backup, false, readErr
			}
			if hasText {
				return text, backup, true, nil
			}
			break
		}
		time.Sleep(15 * time.Millisecond)
	}

	restoreClipboard(backup)
	return "", backup, false, nil
}

func replaceSelection(targetHWND uintptr, newText string, backup clipboardBackup) error {
	if err := writeClipboardText(newText); err != nil {
		return err
	}

	focusWindow(targetHWND)
	time.Sleep(60 * time.Millisecond)

	if err := sendCtrlChord('V'); err != nil {
		return err
	}

	// Allow the target application to read CF_UNICODETEXT before restoring the original clipboard.
	go func(b clipboardBackup) {
		time.Sleep(250 * time.Millisecond)
		restoreClipboard(b)
	}(backup)

	return nil
}

func replaceFieldAll(targetHWND uintptr, newText string) error {
	backup, _ := saveClipboard()
	if err := writeClipboardText(newText); err != nil {
		return err
	}

	focusWindow(targetHWND)
	time.Sleep(60 * time.Millisecond)

	if err := sendCtrlChord('A'); err != nil {
		restoreClipboard(backup)
		return err
	}
	time.Sleep(30 * time.Millisecond)
	if err := sendCtrlChord('V'); err != nil {
		restoreClipboard(backup)
		return err
	}

	go func(b clipboardBackup) {
		time.Sleep(250 * time.Millisecond)
		restoreClipboard(b)
	}(backup)
	return nil
}

func utf16PtrToString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	n, _, _ := procLstrlenW.Call(ptr)
	length := int(n)
	if length <= 0 {
		return ""
	}
	const maxChars = 4 * 1024 * 1024
	if length > maxChars {
		length = maxChars
	}
	buf := make([]uint16, length)
	_, _, _ = procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&buf[0])), ptr, uintptr(length*2))
	return syscall.UTF16ToString(buf)
}

// ReadSelection captures highlighted text from the active window on Windows.
func ReadSelection(ctx context.Context) (string, error) {
	text, _, ok, err := captureSelection()
	if err != nil {
		return "", err
	}
	if !ok || text == "" {
		// Fallback to clipboard content if selection capture was empty
		clip, hasText, err := readClipboardText()
		if err == nil && hasText {
			return clip, nil
		}
	}
	return text, nil
}

// ReplaceSelectedText pastes newText into the active window on Windows.
func ReplaceSelectedText(ctx context.Context, newText string) error {
	hwnd, _, _ := procGetForegroundWindow.Call()
	backup, _ := saveClipboard()
	return replaceSelection(hwnd, newText, backup)
}
