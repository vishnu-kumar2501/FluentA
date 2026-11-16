package openai

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
	"strings"
	"time"

	"grammar/ai"
)

// Provider implements ai.Provider for any OpenAI-compatible API (Groq, DeepSeek, OpenAI, or a proxy worker).
type Provider struct {
	cfg        Config
	httpClient *http.Client
}

var _ ai.Provider = (*Provider)(nil)

// NewProvider creates a dedicated OpenAI-compatible cloud provider.
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
			Timeout:   resolved.RequestTimeout,
		},
	}, nil
}

// Name returns the provider identifier ("cloud").
func (p *Provider) Name() string {
	return "cloud"
}

// Config returns the current configuration.
func (p *Provider) Config() Config {
	return p.cfg
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// HealthCheck verifies the endpoint is reachable.
func (p *Provider) HealthCheck(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	checkCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	// Perform a quick lightweight ping to /models or an empty completion probe
	reqURL := p.cfg.BaseURL + "/models"
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ai.ErrProviderUnavailable, err)
	}

	p.applyHeaders(req)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: could not reach AI cloud endpoint (%v)", ai.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("authentication error: invalid or missing API key (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("AI cloud service temporarily unavailable (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// Chat performs a non-streaming completion.
func (p *Provider) Chat(ctx context.Context, req ai.ChatRequest) (ai.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return ai.ChatResponse{}, fmt.Errorf("%w: at least one message is required", ai.ErrInvalidRequest)
	}

	model := p.cfg.Model
	if strings.TrimSpace(req.Model) != "" {
		model = strings.TrimSpace(req.Model)
	}

	openAIMsgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		openAIMsgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	payload := openAIChatRequest{
		Model:       model,
		Messages:    openAIMsgs,
		Temperature: req.Temperature,
		Stream:      false,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return ai.ChatResponse{}, err
	}

	chatURL := p.cfg.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return ai.ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	p.applyHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ai.ChatResponse{}, fmt.Errorf("%w: %v", ai.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ai.ChatResponse{}, fmt.Errorf("cloud AI error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return ai.ChatResponse{}, fmt.Errorf("decode cloud AI response: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return ai.ChatResponse{}, fmt.Errorf("cloud AI error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return ai.ChatResponse{}, errors.New("cloud AI returned empty choices")
	}

	return ai.ChatResponse{
		Model: model,
		Message: ai.Message{
			Role:    chatResp.Choices[0].Message.Role,
			Content: chatResp.Choices[0].Message.Content,
		},
		DoneReason: chatResp.Choices[0].FinishReason,
	}, nil
}

// ChatStream performs token-by-token streaming completion using Server-Sent Events.
func (p *Provider) ChatStream(ctx context.Context, req ai.ChatRequest, onChunk ai.StreamHandler) (ai.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return ai.ChatResponse{}, fmt.Errorf("%w: at least one message is required", ai.ErrInvalidRequest)
	}
	if onChunk == nil {
		return p.Chat(ctx, req)
	}

	model := p.cfg.Model
	if strings.TrimSpace(req.Model) != "" {
		model = strings.TrimSpace(req.Model)
	}

	openAIMsgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		openAIMsgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	payload := openAIChatRequest{
		Model:       model,
		Messages:    openAIMsgs,
		Temperature: req.Temperature,
		Stream:      true,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return ai.ChatResponse{}, err
	}

	chatURL := p.cfg.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return ai.ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	p.applyHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ai.ChatResponse{}, fmt.Errorf("%w: %v", ai.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ai.ChatResponse{}, fmt.Errorf("cloud AI error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	var accumulated strings.Builder
	var lastFinishReason string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue // SSE keep-alive ping
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Error != nil && chunk.Error.Message != "" {
			return ai.ChatResponse{}, fmt.Errorf("stream chunk error: %s", chunk.Error.Message)
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				accumulated.WriteString(content)
				if err := onChunk(ai.StreamChunk{
					Model:   model,
					Content: content,
				}); err != nil {
					return ai.ChatResponse{}, err
				}
			}
			if chunk.Choices[0].FinishReason != "" {
				lastFinishReason = chunk.Choices[0].FinishReason
			}
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return ai.ChatResponse{}, fmt.Errorf("cloud stream interrupted: %w", err)
	}

	_ = onChunk(ai.StreamChunk{
		Model:      model,
		Done:       true,
		DoneReason: lastFinishReason,
	})

	return ai.ChatResponse{
		Model: model,
		Message: ai.Message{
			Role:    ai.RoleAssistant,
			Content: accumulated.String(),
		},
		DoneReason: lastFinishReason,
	}, nil
}

func (p *Provider) applyHeaders(req *http.Request) {
	if p.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
	req.Header.Set("User-Agent", "FluentA-Desktop/1.0")
}
