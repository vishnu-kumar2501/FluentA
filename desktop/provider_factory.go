package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"grammar/ai"
	"grammar/ai/ollama"
	"grammar/ai/openai"
)

// ResolveAPIKey discovers an API key from Settings, environment, or local groq.txt.
// In production with the Cloudflare proxy, a client API key is not required.
func ResolveAPIKey(customKey string) string {
	if trimmed := strings.TrimSpace(customKey); trimmed != "" {
		return trimmed
	}
	if env := strings.TrimSpace(os.Getenv("GROQ_API_KEY")); env != "" {
		return env
	}
	if env := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); env != "" {
		return env
	}

	// Check for optional groq.txt in working directory or AppData directory or executable directory
	for _, candidate := range []string{
		"groq.txt",
		filepath.Join(AppDataDir(), "groq.txt"),
	} {
		if data, err := os.ReadFile(candidate); err == nil {
			if key := strings.TrimSpace(string(data)); key != "" {
				return key
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		exeKey := filepath.Join(filepath.Dir(exe), "groq.txt")
		if data, err := os.ReadFile(exeKey); err == nil {
			if key := strings.TrimSpace(string(data)); key != "" {
				return key
			}
		}
	}

	return ""
}

// BuildProvider instantiates the appropriate ai.Provider based on the given Settings.
func BuildProvider(s Settings) (ai.Provider, error) {
	norm := NormalizeSettings(s)
	switch norm.AIProviderType {
	case "ollama":
		return ollama.NewProvider(ollama.Config{
			BaseURL:        norm.AIBaseURL,
			Model:          norm.AIModel,
			KeepAlive:      "30m",
			RequestTimeout: ollama.DefaultRequestTimeout,
		})
	case "cloud":
		fallthrough
	default:
		apiKey := ResolveAPIKey(norm.AIApiKey)
		return openai.NewProvider(openai.Config{
			BaseURL:        norm.AIBaseURL,
			Model:          norm.AIModel,
			APIKey:         apiKey,
			RequestTimeout: openai.DefaultRequestTimeout,
		})
	}
}

// TestProviderConnection performs a quick health check on an arbitrary provider configuration.
func TestProviderConnection(ctx context.Context, providerType, baseURL, model, apiKey string) error {
	s := NormalizeSettings(Settings{
		AIProviderType: providerType,
		AIBaseURL:      baseURL,
		AIModel:        model,
		AIApiKey:       apiKey,
	})

	prov, err := BuildProvider(s)
	if err != nil {
		return fmt.Errorf("invalid AI configuration: %w", err)
	}

	checkCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	if err := prov.HealthCheck(checkCtx); err != nil {
		return err
	}
	return nil
}
