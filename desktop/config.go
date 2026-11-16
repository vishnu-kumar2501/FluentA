package desktop

import (
	"time"

	"grammar/grammar"
)

// Config configures the desktop hotkey and background watcher assistant.
type Config struct {
	Hotkey        string
	DefaultMode   grammar.Mode
	ModelName     string
	WatchEnabled  bool
	WatchDebounce time.Duration
	WatchDebug    bool
}
