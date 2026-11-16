package grammar

import (
	"context"
	"strings"
	"testing"

	"grammar/ai"
)

func TestWrapInputText(t *testing.T) {
	input := "Give me a long essay to check."
	wrapped := WrapInputText(input)
	expected := "<text>\nGive me a long essay to check.\n</text>"
	if wrapped != expected {
		t.Fatalf("expected %q, got %q", expected, wrapped)
	}
}

func TestCleanOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain clean text",
			input:    "Give me a long essay to check.",
			expected: "Give me a long essay to check.",
		},
		{
			name:     "wrapped in text tags",
			input:    "<text>\nGive me a long essay to check.\n</text>",
			expected: "Give me a long essay to check.",
		},
		{
			name:     "wrapped in text tags without newlines",
			input:    "<text>Give me a long essay to check.</text>",
			expected: "Give me a long essay to check.",
		},
		{
			name:     "wrapped in markdown code fences",
			input:    "```markdown\nGive me a long essay to check.\n```",
			expected: "Give me a long essay to check.",
		},
		{
			name:     "wrapped in text fences",
			input:    "```\nGive me a long essay to check.\n```",
			expected: "Give me a long essay to check.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanOutput(tt.input)
			if got != tt.expected {
				t.Errorf("CleanOutput(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizingStreamHandler(t *testing.T) {
	tests := []struct {
		name          string
		originalInput string
		chunks        []string
		expected      string
	}{
		{
			name:          "normal stream chunks",
			originalInput: "give me a long esay for check",
			chunks:        []string{"Give ", "me ", "a ", "long ", "essay ", "to ", "check."},
			expected:      "Give me a long essay to check.",
		},
		{
			name:          "stream starting with text tag and ending with closing tag",
			originalInput: "give me a long esay for check",
			chunks:        []string{"<text>", "\nGive ", "me a ", "long essay ", "to check.", "</text>"},
			expected:      "Give me a long essay to check.",
		},
		{
			name:          "stream starting with markdown code fence",
			originalInput: "give me a long esay for check",
			chunks:        []string{"```text\n", "Give me a ", "long essay ", "to check.\n", "```"},
			expected:      "Give me a long essay to check.\n```",
		},
		{
			name:          "short text flushed on done",
			originalInput: "hi",
			chunks:        []string{"Hi"},
			expected:      "Hi",
		},
		{
			name:          "original input contained text tag",
			originalInput: "Here is <text> tag in user code",
			chunks:        []string{"Here is <text> tag in user code"},
			expected:      "Here is <text> tag in user code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			handler := newSanitizingStreamHandler(tt.originalInput, func(chunk ai.StreamChunk) error {
				out.WriteString(chunk.Content)
				return nil
			})

			for i, c := range tt.chunks {
				isLast := i == len(tt.chunks)-1
				err := handler(ai.StreamChunk{
					Content: c,
					Done:    isLast,
				})
				if err != nil {
					t.Fatalf("unexpected stream error: %v", err)
				}
			}

			if got := out.String(); got != tt.expected {
				t.Errorf("stream output mismatch: got %q, expected %q", got, tt.expected)
			}
		})
	}
}

type mockAIProvider struct {
	lastRequest ai.ChatRequest
	response    string
}

func (m *mockAIProvider) Name() string { return "mock" }
func (m *mockAIProvider) HealthCheck(ctx context.Context) error { return nil }
func (m *mockAIProvider) Chat(ctx context.Context, req ai.ChatRequest) (ai.ChatResponse, error) {
	m.lastRequest = req
	return ai.ChatResponse{
		Message: ai.Message{
			Role:    ai.RoleAssistant,
			Content: m.response,
		},
	}, nil
}
func (m *mockAIProvider) ChatStream(ctx context.Context, req ai.ChatRequest, onChunk ai.StreamHandler) (ai.ChatResponse, error) {
	m.lastRequest = req
	if onChunk != nil {
		_ = onChunk(ai.StreamChunk{Content: m.response, Done: true})
	}
	return ai.ChatResponse{
		Message: ai.Message{
			Role:    ai.RoleAssistant,
			Content: m.response,
		},
	}, nil
}

func TestChecker_CheckWithOptionsStream_WrapsInputAndCleansOutput(t *testing.T) {
	mock := &mockAIProvider{
		response: "<text>\nGive me a long essay to check.\n</text>",
	}
	checker, err := NewChecker(mock)
	if err != nil {
		t.Fatalf("NewChecker failed: %v", err)
	}

	var streamed strings.Builder
	resp, err := checker.CheckWithOptionsStream(context.Background(), "give me a long esay for check", CheckOptions{
		Mode: ModeFix,
	}, func(sc ai.StreamChunk) error {
		streamed.WriteString(sc.Content)
		return nil
	})
	if err != nil {
		t.Fatalf("CheckWithOptionsStream failed: %v", err)
	}

	// Verify request wrapping
	if len(mock.lastRequest.Messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(mock.lastRequest.Messages))
	}
	userMsg := mock.lastRequest.Messages[1].Content
	if !strings.HasPrefix(userMsg, "<text>\n") || !strings.HasSuffix(userMsg, "\n</text>") {
		t.Errorf("expected user message to be wrapped in <text>, got %q", userMsg)
	}

	// Verify system prompt anti-instruction directives
	sysMsg := mock.lastRequest.Messages[0].Content
	if !strings.Contains(sysMsg, "CRITICAL INSTRUCTIONS") || !strings.Contains(sysMsg, "RAW PASSIVE INPUT") {
		t.Errorf("expected system prompt to have critical anti-instruction directives, got: %s", sysMsg)
	}

	// Verify output cleaning (both stream and final resp)
	expectedClean := "Give me a long essay to check."
	if resp.Message.Content != expectedClean {
		t.Errorf("expected resp.Message.Content = %q, got %q", expectedClean, resp.Message.Content)
	}
	if streamed.String() != expectedClean {
		t.Errorf("expected streamed content = %q, got %q", expectedClean, streamed.String())
	}
}
