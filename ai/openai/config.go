package openai

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"grammar/ai"
)

const (
	// DefaultBaseURL is the OpenAI-compatible API base URL (Groq by default).
	DefaultBaseURL = "https://api.groq.com/openai/v1"
	// DefaultModel is the default high-performance model on Groq.
	DefaultModel = "openai/gpt-oss-120b"
	// DefaultRequestTimeout is the default per-request timeout.
	DefaultRequestTimeout = 60 * time.Second
	// DefaultDialTimeout is the default TCP dial timeout.
	DefaultDialTimeout = 5 * time.Second
)

// Config configures an OpenAI-compatible API provider (Groq, DeepSeek, OpenAI, Cloudflare proxy, etc.).
type Config struct {
	BaseURL        string
	Model          string
	APIKey         string
	RequestTimeout time.Duration
	DialTimeout    time.Duration
}

// DefaultConfig returns a default Config using Groq's endpoint and Llama 3.3.
func DefaultConfig() Config {
	return Config{
		BaseURL:        DefaultBaseURL,
		Model:          DefaultModel,
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

	c.APIKey = strings.TrimSpace(c.APIKey)

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
