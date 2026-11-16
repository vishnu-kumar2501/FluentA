//go:build windows

package desktop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName    = "FluentA"
)

func executableCommand() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	p = filepath.Clean(p)
	return fmt.Sprintf("%q", p), nil
}

// IsAutostartEnabled checks if FluentA is registered under HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
func IsAutostartEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer k.Close()

	val, _, err := k.GetStringValue(runValueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(val) != "", nil
}

// SetAutostartEnabled registers or removes FluentA from HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
func SetAutostartEnabled(enabled bool) error {
	if enabled {
		k, err := registry.OpenKey(registry.CURRENT_USER, runRegistryPath, registry.SET_VALUE)
		if err != nil {
			return err
		}
		defer k.Close()

		cmd, err := executableCommand()
		if err != nil {
			return err
		}
		return k.SetStringValue(runValueName, cmd)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, runRegistryPath, registry.SET_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	defer k.Close()

	err = k.DeleteValue(runValueName)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

