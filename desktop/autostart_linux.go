//go:build linux

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
)

func autostartFilePath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "autostart", "fluenta.desktop"), nil
}

// IsAutostartEnabled checks if ~/.config/autostart/fluenta.desktop exists.
func IsAutostartEnabled() (bool, error) {
	p, err := autostartFilePath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// SetAutostartEnabled creates or removes ~/.config/autostart/fluenta.desktop.
func SetAutostartEnabled(enabled bool) error {
	p, err := autostartFilePath()
	if err != nil {
		return err
	}

	if enabled {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=FluentA
Comment=AI Writing & Grammar Assistant
Exec=%s
Icon=fluenta
Terminal=false
Categories=Utility;Office;
X-GNOME-Autostart-enabled=true
`, exe)
		return os.WriteFile(p, []byte(content), 0644)
	}

	err = os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
