package desktop

import (
	"fmt"
	"strings"
)

// Rect mirrors Win32 RECT coordinates.
type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// Win32 RegisterHotKey modifier constants.
const (
	ModAlt      uint32 = 0x0001
	ModControl  uint32 = 0x0002
	ModShift    uint32 = 0x0004
	ModWin      uint32 = 0x0008
	ModNoRepeat uint32 = 0x4000
)

// ClampPopupPosition positions a popup near (cursorX, cursorY) while keeping it inside workArea.
func ClampPopupPosition(cursorX, cursorY, width, height int32, workArea Rect) (int32, int32) {
	const offsetX, offsetY int32 = 12, 16
	x := cursorX + offsetX
	y := cursorY + offsetY

	if x+width > workArea.Right {
		x = cursorX - width - offsetX
	}
	if y+height > workArea.Bottom {
		y = cursorY - height - offsetY
	}
	if x+width > workArea.Right {
		x = workArea.Right - width
	}
	if y+height > workArea.Bottom {
		y = workArea.Bottom - height
	}
	if x < workArea.Left {
		x = workArea.Left
	}
	if y < workArea.Top {
		y = workArea.Top
	}
	return x, y
}

// ParseHotkey parses a shortcut such as "ctrl+shift+g" into Win32 RegisterHotKey modifiers and virtual key code.
func ParseHotkey(spec string) (modifiers uint32, vk uint32, normalized string, err error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return 0, 0, "", fmt.Errorf("hotkey must not be empty")
	}

	parts := strings.Split(trimmed, "+")
	var modNames []string
	var keyName string

	for _, raw := range parts {
		part := strings.ToLower(strings.TrimSpace(raw))
		if part == "" {
			return 0, 0, "", fmt.Errorf("invalid hotkey %q", spec)
		}
		switch part {
		case "ctrl", "control":
			modifiers |= ModControl
			modNames = append(modNames, "Ctrl")
		case "shift":
			modifiers |= ModShift
			modNames = append(modNames, "Shift")
		case "alt":
			modifiers |= ModAlt
			modNames = append(modNames, "Alt")
		case "win", "super":
			modifiers |= ModWin
			modNames = append(modNames, "Win")
		default:
			if keyName != "" {
				return 0, 0, "", fmt.Errorf("hotkey %q has multiple primary keys", spec)
			}
			keyName = part
		}
	}

	if modifiers == 0 {
		return 0, 0, "", fmt.Errorf("hotkey %q must include at least one modifier (ctrl, shift, alt)", spec)
	}
	if keyName == "" {
		return 0, 0, "", fmt.Errorf("hotkey %q is missing a primary key", spec)
	}

	vkCode, displayKey, err := parseVirtualKey(keyName)
	if err != nil {
		return 0, 0, "", err
	}

	modifiers |= ModNoRepeat
	return modifiers, vkCode, strings.Join(append(modNames, displayKey), "+"), nil
}

func parseVirtualKey(key string) (uint32, string, error) {
	if len(key) == 1 {
		ch := key[0]
		if ch >= 'a' && ch <= 'z' {
			upper := ch - 'a' + 'A'
			return uint32(upper), string(upper), nil
		}
		if ch >= '0' && ch <= '9' {
			return uint32(ch), string(ch), nil
		}
	}
	if key == "space" {
		return 0x20, "Space", nil
	}
	if strings.HasPrefix(key, "f") && len(key) >= 2 && len(key) <= 3 {
		var n int
		if _, err := fmt.Sscanf(key, "f%d", &n); err == nil && n >= 1 && n <= 12 {
			return uint32(0x70 + n - 1), strings.ToUpper(key), nil
		}
	}
	return 0, "", fmt.Errorf("unsupported hotkey key %q (use A-Z, 0-9, Space, or F1-F12)", key)
}

func toCRLF(s string) string {
	normalized := strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(normalized, "\n", "\r\n")
}

func toLF(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// ClampBadgePosition places the floating watcher pill near the bottom-right inside or beside fieldBounds.
func ClampBadgePosition(fieldBounds Rect, badgeW, badgeH int32, workArea Rect) (int32, int32) {
	const margin int32 = 8
	if fieldBounds.Right <= fieldBounds.Left || fieldBounds.Bottom <= fieldBounds.Top {
		return workArea.Right - badgeW - 16, workArea.Bottom - badgeH - 16
	}

	x := fieldBounds.Right - badgeW - margin
	y := fieldBounds.Bottom - badgeH - margin

	// For short single-line inputs, place the badge just below/right of the field so it doesn't cover text.
	if (fieldBounds.Bottom - fieldBounds.Top) < (badgeH + 18) {
		y = fieldBounds.Bottom + 4
	}

	if x+badgeW > workArea.Right {
		x = workArea.Right - badgeW
	}
	if y+badgeH > workArea.Bottom {
		y = fieldBounds.Top - badgeH - 4
	}
	if y+badgeH > workArea.Bottom {
		y = workArea.Bottom - badgeH
	}
	if x < workArea.Left {
		x = workArea.Left
	}
	if y < workArea.Top {
		y = workArea.Top
	}
	return x, y
}

// ShouldWatchText reports whether text captured from a focused control should be queued for grammar checking.
func ShouldWatchText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 8 {
		return false
	}
	return strings.ContainsAny(trimmed, " \t\n")
}

