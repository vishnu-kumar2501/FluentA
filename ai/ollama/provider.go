package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"grammar/ai"
)

const (
	chatEndpoint = "/api/chat"
	tagsEndpoint = "/api/tags"
	maxErrorBody = 8192
)

// Provider implements ai.Provider against a local Ollama instance.
type Provider struct {
	cfg        Config
	httpClient *http.Client
}

// Compile-time verification that Provider satisfies ai.Provider.
var _ ai.Provider = (*Provider)(nil)

// NewProvider creates a dedicated Ollama provider with the given configuration.
func NewProvider(cfg Config) (*Provider, error) {
	resolved := cfg.withDefaults()
	if err := resolved.validate(); err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout: resolved.DialTimeout,
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Provider{
		cfg: resolved,
		httpClient: &http.Client{
			Transport: transport,
		},
	}, nil
}

// Name returns the provider identifier ("ollama").
func (p *Provider) Name() string {
	return "ollama"
}

// Config returns a copy of the resolved provider configuration.
func (p *Provider) Config() Config {
	return p.cfg
}

// HealthCheck verifies Ollama is reachable and the configured model is installed.
func (p *Provider) HealthCheck(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.BaseURL+tagsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("%w: build health check request: %v", ai.ErrProviderUnavailable, err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return classifyTransportError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readAPIError(resp)
	}

	var tags tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return fmt.Errorf("%w: decode tags response: %v", ai.ErrProviderUnavailable, err)
	}

	if !hasModel(tags.Models, p.cfg.Model) {
		return fmt.Errorf("%w: model %q is not installed in Ollama", ai.ErrModelNotFound, p.cfg.Model)
	}

	return nil
}

// Chat executes inference using Ollama streaming responses and returns the final assembled response.
func (p *Provider) Chat(ctx context.Context, req ai.ChatRequest) (ai.ChatResponse, error) {
	return p.ChatStream(ctx, req, nil)
}

// ChatStream sends a streaming /api/chat request to Ollama and invokes onChunk for each streamed chunk.
func (p *Provider) ChatStream(ctx context.Context, req ai.ChatRequest, onChunk ai.StreamHandler) (ai.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return ai.ChatResponse{}, fmt.Errorf("%w: messages must not be empty", ai.ErrInvalidRequest)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.cfg.Model
	}

	keepAlive := strings.TrimSpace(req.KeepAlive)
	if keepAlive == "" {
		keepAlive = p.cfg.KeepAlive
	}

	think := p.cfg.Think
	if req.Think != nil {
		think = req.Think
	}

	payload := chatAPIRequest{
		Model:     model,
		Messages:  req.Messages,
		Stream:    true,
		KeepAlive: encodeKeepAlive(keepAlive),
		Think:     think,
		Format:    req.Format,
	}
	if req.Temperature != nil {
		payload.Options = &chatOptions{Temperature: req.Temperature}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return ai.ChatResponse{}, fmt.Errorf("%w: marshal request: %v", ai.ErrInvalidRequest, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+chatEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return ai.ChatResponse{}, fmt.Errorf("%w: build chat request: %v", ai.ErrInvalidRequest, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/x-ndjson")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ai.ChatResponse{}, classifyTransportError(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ai.ChatResponse{}, readAPIError(resp)
	}

	return readStreamResponse(ctx, resp.Body, model, onChunk)
}

func (p *Provider) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if p.cfg.RequestTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, p.cfg.RequestTimeout)
}

func readStreamResponse(ctx context.Context, body io.Reader, requestedModel string, onChunk ai.StreamHandler) (ai.ChatResponse, error) {
	reader := bufio.NewReaderSize(body, 64*1024)
	var content strings.Builder
	var thinking strings.Builder
	finalModel := requestedModel
	var doneReason string
	var completed bool

	for {
		if err := ctx.Err(); err != nil {
			return ai.ChatResponse{}, err
		}

		line, readErr := reader.ReadBytes('\n')
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			var apiChunk chatAPIStreamChunk
			if err := json.Unmarshal(trimmed, &apiChunk); err != nil {
				return ai.ChatResponse{}, fmt.Errorf("%w: decode stream chunk: %v", ai.ErrProviderUnavailable, err)
			}
			if apiChunk.Error != "" {
				return ai.ChatResponse{}, classifyAPIErrorMessage(http.StatusInternalServerError, apiChunk.Error)
			}
			if apiChunk.Model != "" {
				finalModel = apiChunk.Model
			}
			if apiChunk.Message.Content != "" {
				content.WriteString(apiChunk.Message.Content)
			}
			if apiChunk.Message.Thinking != "" {
				thinking.WriteString(apiChunk.Message.Thinking)
			}
			if apiChunk.Done {
				completed = true
				doneReason = apiChunk.DoneReason
			}

			if onChunk != nil {
				chunk := ai.StreamChunk{
					Model:      finalModel,
					Content:    apiChunk.Message.Content,
					Thinking:   apiChunk.Message.Thinking,
					Done:       apiChunk.Done,
					DoneReason: apiChunk.DoneReason,
				}
				if err := onChunk(chunk); err != nil {
					return ai.ChatResponse{}, err
				}
			}

			if apiChunk.Done {
				break
			}
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ai.ChatResponse{}, ctxErr
			}
			return ai.ChatResponse{}, fmt.Errorf("%w: read stream: %v", ai.ErrProviderUnavailable, readErr)
		}
	}

	if !completed {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ai.ChatResponse{}, ctxErr
		}
		return ai.ChatResponse{}, fmt.Errorf("%w: stream ended before completion", ai.ErrProviderUnavailable)
	}

	return ai.ChatResponse{
		Model: finalModel,
		Message: ai.Message{
			Role:    ai.RoleAssistant,
			Content: content.String(),
		},
		Thinking:   thinking.String(),
		DoneReason: doneReason,
	}, nil
}

func classifyTransportError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %v", ai.ErrProviderUnavailable, err)
}

func readAPIError(resp *http.Response) error {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	if err != nil {
		return fmt.Errorf("%w: status %d (failed reading error body: %v)", ai.ErrProviderUnavailable, resp.StatusCode, err)
	}

	var errPayload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &errPayload) == nil && strings.TrimSpace(errPayload.Error) != "" {
		return classifyAPIErrorMessage(resp.StatusCode, errPayload.Error)
	}

	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return classifyAPIErrorMessage(resp.StatusCode, msg)
}

func classifyAPIErrorMessage(status int, message string) error {
	lower := strings.ToLower(message)
	if status == http.StatusNotFound || (strings.Contains(lower, "model") && strings.Contains(lower, "not found")) {
		return fmt.Errorf("%w: %s", ai.ErrModelNotFound, message)
	}
	return fmt.Errorf("%w: ollama error (status %d): %s", ai.ErrProviderUnavailable, status, message)
}

func hasModel(models []tagModel, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	withLatest := target
	if !strings.Contains(target, ":") {
		withLatest = target + ":latest"
	}
	for _, m := range models {
		if m.Name == target || m.Model == target || m.Name == withLatest || m.Model == withLatest {
			return true
		}
	}
	return false
}

func encodeKeepAlive(value string) any {
	if value == "" {
		return nil
	}
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return n
	}
	return value
}

type tagsResponse struct {
	Models []tagModel `json:"models"`
}

type tagModel struct {
	Name  string `json:"name"`
	Model string `json:"model"`
}

type chatAPIRequest struct {
	Model     string       `json:"model"`
	Messages  []ai.Message `json:"messages"`
	Stream    bool         `json:"stream"`
	KeepAlive any          `json:"keep_alive,omitempty"`
	Think     *bool        `json:"think,omitempty"`
	Format    any          `json:"format,omitempty"`
	Options   *chatOptions `json:"options,omitempty"`
}

type chatOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
}

type chatAPIStreamChunk struct {
	Model      string `json:"model"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
	Error      string `json:"error"`
	Message    struct {
		Role     string `json:"role"`
		Content  string `json:"content"`
		Thinking string `json:"thinking"`
	} `json:"message"`
}

