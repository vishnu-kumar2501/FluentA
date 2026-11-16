//go:build linux

package desktop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ReadSelection retrieves text from the Linux Primary selection (highlighted text)
// or standard clipboard using native Wayland or X11 tools.
func ReadSelection(ctx context.Context) (string, error) {
	// 1. Try Wayland primary selection (wl-paste -p)
	if isCommandAvailable("wl-paste") {
		if text, err := runOutput(ctx, "wl-paste", "-p", "--no-newline"); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
		// Fallback to standard Wayland clipboard
		if text, err := runOutput(ctx, "wl-paste", "--no-newline"); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
	}

	// 2. Try X11 primary selection (xclip)
	if isCommandAvailable("xclip") {
		if text, err := runOutput(ctx, "xclip", "-o", "-selection", "primary"); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
		// Fallback to standard X11 clipboard
		if text, err := runOutput(ctx, "xclip", "-o", "-selection", "clipboard"); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
	}

	// 3. Try xsel
	if isCommandAvailable("xsel") {
		if text, err := runOutput(ctx, "xsel", "-o", "-p"); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
		if text, err := runOutput(ctx, "xsel", "-o", "-b"); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
	}

	return "", errors.New("no Linux clipboard or selection utility found (please install wl-clipboard, xclip, or xsel)")
}

// WriteClipboard writes text into both the Linux clipboard and primary selection.
func WriteClipboard(ctx context.Context, text string) error {
	var written bool

	if isCommandAvailable("wl-copy") {
		// Wayland
		cmd := exec.CommandContext(ctx, "wl-copy")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			written = true
			// Also write to primary selection
			cmdPrimary := exec.CommandContext(ctx, "wl-copy", "-p")
			cmdPrimary.Stdin = strings.NewReader(text)
			_ = cmdPrimary.Run()
		}
	}

	if isCommandAvailable("xclip") {
		// X11 clipboard
		cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			written = true
			// Also write to primary selection
			cmdPrimary := exec.CommandContext(ctx, "xclip", "-selection", "primary")
			cmdPrimary.Stdin = strings.NewReader(text)
			_ = cmdPrimary.Run()
		}
	}

	if !written && isCommandAvailable("xsel") {
		cmd := exec.CommandContext(ctx, "xsel", "-b", "-i")
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			written = true
		}
	}

	if !written {
		return errors.New("failed to write to Linux clipboard (wl-copy, xclip, or xsel required)")
	}
	return nil
}

// ReplaceSelectedText writes the new text to clipboard and triggers a paste keystroke (Ctrl+V)
// in the active window.
func ReplaceSelectedText(ctx context.Context, newText string) error {
	if err := WriteClipboard(ctx, newText); err != nil {
		return err
	}

	// Small pause to allow clipboard synchronization
	time.Sleep(30 * time.Millisecond)

	// Trigger Ctrl+V
	if isCommandAvailable("xdotool") {
		return exec.CommandContext(ctx, "xdotool", "key", "--clearmodifiers", "ctrl+v").Run()
	}
	if isCommandAvailable("wtype") {
		return exec.CommandContext(ctx, "wtype", "-M", "ctrl", "-k", "v", "-m", "ctrl").Run()
	}
	if isCommandAvailable("ydotool") {
		// Keycode 29 = Ctrl, 47 = V
		return exec.CommandContext(ctx, "ydotool", "key", "29:1", "47:1", "47:0", "29:0").Run()
	}

	return fmt.Errorf("text copied to clipboard, but automated paste requires 'xdotool' (X11) or 'wtype'/'ydotool' (Wayland)")
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}
