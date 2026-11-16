package ollama

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"grammar/ai"
)

const (
	// DefaultBaseURL is the local Ollama API base URL.
	DefaultBaseURL = "http://127.0.0.1:11434"
	// DefaultModel is the configured local Ollama model.
	DefaultModel = "qwen3.5:4b"
	// DefaultKeepAlive keeps the model loaded in memory between requests.
	DefaultKeepAlive = "10m"
	// DefaultRequestTimeout is the default per-request timeout.
	DefaultRequestTimeout = 2 * time.Minute
	// DefaultDialTimeout is the default TCP connection timeout.
	DefaultDialTimeout = 5 * time.Second
)

// Config configures the local Ollama inference provider.
type Config struct {
	// BaseURL is the Ollama server URL (defaults to http://127.0.0.1:11434).
	BaseURL string
	// Model is the installed Ollama model name (defaults to qwen3.5:4b).
	Model string
	// KeepAlive controls how long Ollama keeps the model loaded in memory (defaults to 10m).
	// Examples: "10m", "1h", "0" (unload immediately), "-1" (keep loaded indefinitely).
	KeepAlive string
	// RequestTimeout sets an upper bound for each API call when positive.
	// Caller context deadlines still apply if they are shorter.
	RequestTimeout time.Duration
	// DialTimeout sets the TCP dial timeout to the Ollama server.
	DialTimeout time.Duration
	// Think optionally configures whether thinking tokens are requested by default.
	Think *bool
}

// DefaultConfig returns a Config using the local Ollama settings.
func DefaultConfig() Config {
	return Config{
		BaseURL:        DefaultBaseURL,
		Model:          DefaultModel,
		KeepAlive:      DefaultKeepAlive,
		RequestTimeout: DefaultRequestTimeout,
		DialTimeout:    DefaultDialTimeout,
	}
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")

	if strings.TrimSpace(c.Model) == "" {
		c.Model = DefaultModel
	} else {
		c.Model = strings.TrimSpace(c.Model)
	}

	if strings.TrimSpace(c.KeepAlive) == "" {
		c.KeepAlive = DefaultKeepAlive
	} else {
		c.KeepAlive = strings.TrimSpace(c.KeepAlive)
	}

	if c.RequestTimeout == 0 {
		c.RequestTimeout = DefaultRequestTimeout
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = DefaultDialTimeout
	}
	return c
}

func (c Config) validate() error {
	parsed, err := url.Parse(c.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%w: invalid BaseURL %q", ai.ErrInvalidConfig, c.BaseURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: unsupported URL scheme %q", ai.ErrInvalidConfig, parsed.Scheme)
	}
	if c.Model == "" {
		return fmt.Errorf("%w: model must not be empty", ai.ErrInvalidConfig)
	}
	if c.RequestTimeout < 0 {
		return fmt.Errorf("%w: RequestTimeout must not be negative", ai.ErrInvalidConfig)
	}
	if c.DialTimeout < 0 {
		return fmt.Errorf("%w: DialTimeout must not be negative", ai.ErrInvalidConfig)
	}
	return nil
}

