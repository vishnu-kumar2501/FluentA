package ai

import (
	"context"
	"errors"
)

var (
	// ErrProviderUnavailable indicates the AI runtime cannot be reached or is not running.
	ErrProviderUnavailable = errors.New("ai provider unavailable")
	// ErrModelNotFound indicates the configured model is not available on the runtime.
	ErrModelNotFound = errors.New("ai model not found")
	// ErrInvalidRequest indicates the chat request is invalid (for example, empty messages).
	ErrInvalidRequest = errors.New("invalid ai request")
	// ErrInvalidConfig indicates the provider configuration is invalid.
	ErrInvalidConfig = errors.New("invalid ai provider configuration")
)

// Role constants for chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message represents a single turn in a chat conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest describes an inference request sent to an AI Provider.
type ChatRequest struct {
	// Messages is the ordered conversation history. Must be non-empty.
	Messages []Message
	// Model optionally overrides the provider's configured default model.
	Model string
	// Temperature optionally overrides sampling temperature.
	Temperature *float64
	// KeepAlive optionally overrides how long the runtime keeps the model loaded.
	KeepAlive string
	// Think optionally enables or disables model thinking output when supported.
	Think *bool
	// Format optionally requests a structured response format (such as "json").
	Format any
}

// StreamChunk represents an incremental chunk emitted during streaming inference.
type StreamChunk struct {
	Model      string
	Content    string
	Thinking   string
	Done       bool
	DoneReason string
}

// ChatResponse represents the completed result of a chat inference call.
type ChatResponse struct {
	Model      string
	Message    Message
	Thinking   string
	DoneReason string
}

// StreamHandler receives incremental chunks during streaming inference.
// Returning a non-nil error aborts streaming and returns that error.
type StreamHandler func(chunk StreamChunk) error

// Provider abstracts AI inference behind a runtime-agnostic interface
// so the underlying AI runtime can be swapped without changing callers.
type Provider interface {
	// Name returns a human-readable identifier for the provider (e.g. "ollama").
	Name() string
	// HealthCheck verifies the AI runtime is reachable and the configured model is available.
	HealthCheck(ctx context.Context) error
	// Chat performs inference and returns the complete response.
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	// ChatStream performs streaming inference, invoking onChunk as tokens arrive.
	ChatStream(ctx context.Context, req ChatRequest, onChunk StreamHandler) (ChatResponse, error)
}

