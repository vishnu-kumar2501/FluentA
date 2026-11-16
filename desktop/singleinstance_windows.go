//go:build windows

package desktop

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"syscall"
	"unsafe"
)

// ErrAlreadyRunning is returned when another FluentA instance owns the instance lock.
var ErrAlreadyRunning = errors.New("fluentA is already running")

type instanceLock struct {
	handle uintptr
}

func (l *instanceLock) Release() {
	if l == nil || l.handle == 0 {
		return
	}
	_, _, _ = procCloseHandle.Call(l.handle)
	l.handle = 0
}

func instanceMutexName(dataDir string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(dataDir))))
	return fmt.Sprintf("Local\\FluentA_%08x", h.Sum32())
}

func acquireSingleInstance(name string) (*instanceLock, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("mutex name must not be empty")
	}
	handle, _, callErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16Ptr(name))))
	if handle == 0 {
		return nil, fmt.Errorf("CreateMutexW(%s): %w", name, callErr)
	}
	const errAlreadyExists = syscall.Errno(183)
	if errors.Is(callErr, errAlreadyExists) {
		_, _, _ = procCloseHandle.Call(handle)
		return nil, ErrAlreadyRunning
	}
	return &instanceLock{handle: handle}, nil
}

