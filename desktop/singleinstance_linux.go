//go:build linux

package desktop

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// ErrAlreadyRunning is returned when another FluentA instance owns the instance lock.
var ErrAlreadyRunning = errors.New("fluentA is already running")

type instanceLock struct {
	file *os.File
}

func (l *instanceLock) Release() {
	if l == nil || l.file == nil {
		return
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	_ = l.file.Close()
	l.file = nil
}

func instanceMutexName(dataDir string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(dataDir))))
	return fmt.Sprintf("fluenta_%08x.lock", h.Sum32())
}

func acquireSingleInstance(name string) (*instanceLock, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("lock name must not be empty")
	}

	lockDir := os.TempDir()
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		lockDir = runtimeDir
	}

	lockPath := filepath.Join(lockDir, name)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", lockPath, err)
	}

	// Non-blocking exclusive lock
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("acquire lock on %s: %w", lockPath, err)
	}

	return &instanceLock{file: f}, nil
}
