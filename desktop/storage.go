package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultHistoryMaxItems = 50
	settingsFileName       = "settings.json"
	historyFileName        = "history.json"

	DefaultCloudBaseURL  = "https://fluenta-proxy.kumarvishnu610.workers.dev"
	DefaultCloudModel    = "openai/gpt-oss-120b"
	DefaultOllamaBaseURL = "http://127.0.0.1:11434"
	DefaultOllamaModel   = "qwen3.5:4b"
)

// Settings stores persisted user preferences under %LOCALAPPDATA%\FluentA\settings.json.
type Settings struct {
	WatchEnabled    *bool    `json:"watchEnabled,omitempty"`
	Paused          bool     `json:"paused,omitempty"`
	Hotkey          string   `json:"hotkey,omitempty"`
	LaunchOnStartup *bool    `json:"launchOnStartup,omitempty"`
	DefaultMode     string   `json:"defaultMode,omitempty"`
	Language        string   `json:"language,omitempty"`
	Theme           string   `json:"theme,omitempty"` // "system" | "light" | "dark"
	ExcludedApps    []string `json:"excludedApps,omitempty"`

	// AI Provider configuration
	AIProviderType string `json:"aiProviderType,omitempty"` // "cloud" | "ollama"
	AIBaseURL      string `json:"aiBaseURL,omitempty"`      // e.g. "https://api.groq.com/openai/v1" or custom proxy
	AIModel        string `json:"aiModel,omitempty"`        // e.g. "openai/gpt-oss-120b"
	AIApiKey       string `json:"aiApiKey,omitempty"`       // optional direct API key
}

// HistoryItem records a completed grammar check/rewrite for quick restore.
type HistoryItem struct {
	ID         string `json:"id"`
	Timestamp  string `json:"timestamp"`
	Mode       string `json:"mode"`
	Language   string `json:"language,omitempty"`
	SourceText string `json:"sourceText"`
	OutputText string `json:"outputText"`
}

// DefaultExcludedApps returns executables excluded from background UIA watching by default
// (password managers, terminals, and secret-entry tools).
func DefaultExcludedApps() []string {
	return []string{
		"keepass.exe",
		"keepassxc.exe",
		"1password.exe",
		"bitwarden.exe",
		"lastpass.exe",
		"cmd.exe",
		"powershell.exe",
		"pwsh.exe",
		"windowsterminal.exe",
	}
}

// AppDataDir resolves the writable data directory for settings, history, and logs.
// Tests or portable setups can override it via GRAMMAR_DATA_DIR.
func AppDataDir() string {
	if env := strings.TrimSpace(os.Getenv("GRAMMAR_DATA_DIR")); env != "" {
		return env
	}
	if base := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); base != "" {
		return filepath.Join(base, "FluentA")
	}
	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		return filepath.Join(cfgDir, "FluentA")
	}
	return filepath.Join(os.TempDir(), "FluentA")
}

// NormalizeSettings fills missing defaults and normalizes app exclusion entries to lowercase basenames.
func NormalizeSettings(s Settings) Settings {
	s.Hotkey = strings.TrimSpace(s.Hotkey)
	if strings.TrimSpace(s.DefaultMode) == "" {
		s.DefaultMode = "fix"
	} else {
		s.DefaultMode = strings.ToLower(strings.TrimSpace(s.DefaultMode))
	}
	if strings.TrimSpace(s.Language) == "" {
		s.Language = "Auto-detect"
	}
	switch strings.ToLower(strings.TrimSpace(s.Theme)) {
	case "light", "dark", "system":
		s.Theme = strings.ToLower(strings.TrimSpace(s.Theme))
	default:
		s.Theme = "system"
	}
	if s.ExcludedApps == nil {
		s.ExcludedApps = DefaultExcludedApps()
	} else {
		s.ExcludedApps = NormalizeExcludedApps(s.ExcludedApps)
	}

	s.AIProviderType = strings.ToLower(strings.TrimSpace(s.AIProviderType))
	if s.AIProviderType != "ollama" && s.AIProviderType != "cloud" {
		s.AIProviderType = "cloud"
	}
	s.AIBaseURL = strings.TrimSpace(s.AIBaseURL)
	if s.AIBaseURL == "" || s.AIBaseURL == "https://api.groq.com/openai/v1" {
		if s.AIProviderType == "cloud" {
			s.AIBaseURL = DefaultCloudBaseURL
		} else {
			s.AIBaseURL = DefaultOllamaBaseURL
		}
	}
	s.AIModel = strings.TrimSpace(s.AIModel)
	if s.AIModel == "" || s.AIModel == "llama-3.3-70b-versatile" {
		if s.AIProviderType == "cloud" {
			s.AIModel = DefaultCloudModel
		} else {
			s.AIModel = DefaultOllamaModel
		}
	}
	s.AIApiKey = strings.TrimSpace(s.AIApiKey)
	return s
}

// NormalizeExcludedApps cleans, lowercases, and deduplicates process names while keeping empty lists empty.
func NormalizeExcludedApps(apps []string) []string {
	out := make([]string, 0, len(apps))
	seen := make(map[string]struct{}, len(apps))
	for _, a := range apps {
		clean := strings.ToLower(strings.TrimSpace(filepath.Base(a)))
		if clean == "" || clean == "." {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

// IsAppExcluded reports whether processName (e.g. "keepass.exe" or full path) matches excludedApps.
func IsAppExcluded(processName string, excludedApps []string) bool {
	if strings.TrimSpace(processName) == "" {
		return false
	}
	base := strings.ToLower(strings.TrimSpace(filepath.Base(processName)))
	baseNoExt := strings.TrimSuffix(base, ".exe")
	for _, item := range excludedApps {
		clean := strings.ToLower(strings.TrimSpace(filepath.Base(item)))
		if clean == "" {
			continue
		}
		if clean == base || strings.TrimSuffix(clean, ".exe") == baseNoExt {
			return true
		}
	}
	return false
}

// LoadSettings reads settings.json from dir. Returns (defaults, false, nil) if not present yet.
func LoadSettings(dir string) (Settings, bool, error) {
	if dir == "" {
		dir = AppDataDir()
	}
	path := filepath.Join(dir, settingsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NormalizeSettings(Settings{}), false, nil
		}
		return NormalizeSettings(Settings{}), false, fmt.Errorf("read settings: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return NormalizeSettings(Settings{}), false, fmt.Errorf("parse settings: %w", err)
	}
	return NormalizeSettings(s), true, nil
}

// SaveSettings writes settings.json atomically into dir.
func SaveSettings(dir string, s Settings) error {
	if dir == "" {
		dir = AppDataDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	norm := NormalizeSettings(s)
	data, err := json.MarshalIndent(norm, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return writeFileAtomic(filepath.Join(dir, settingsFileName), data)
}

// LoadHistory reads history.json from dir (returning nil, nil if not created yet).
func LoadHistory(dir string) ([]HistoryItem, error) {
	if dir == "" {
		dir = AppDataDir()
	}
	path := filepath.Join(dir, historyFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read history: %w", err)
	}
	var items []HistoryItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}
	return items, nil
}

// AppendHistory prepends item (skipping immediate duplicates), caps at maxItems, and persists history.json.
func AppendHistory(dir string, item HistoryItem, maxItems int) ([]HistoryItem, error) {
	if maxItems <= 0 {
		maxItems = defaultHistoryMaxItems
	}
	if strings.TrimSpace(item.SourceText) == "" || strings.TrimSpace(item.OutputText) == "" {
		return LoadHistory(dir)
	}
	if item.ID == "" {
		item.ID = fmt.Sprintf("h-%d", time.Now().UnixNano())
	}
	if item.Timestamp == "" {
		item.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	existing, err := LoadHistory(dir)
	if err != nil {
		existing = nil
	}
	if len(existing) > 0 &&
		existing[0].Mode == item.Mode &&
		existing[0].SourceText == item.SourceText &&
		existing[0].OutputText == item.OutputText {
		return existing, nil
	}

	updated := make([]HistoryItem, 0, min(len(existing)+1, maxItems))
	updated = append(updated, item)
	for _, old := range existing {
		if len(updated) >= maxItems {
			break
		}
		updated = append(updated, old)
	}

	if err := saveHistory(dir, updated); err != nil {
		return existing, err
	}
	return updated, nil
}

// ClearHistory removes all stored history entries in dir.
func ClearHistory(dir string) error {
	if dir == "" {
		dir = AppDataDir()
	}
	return saveHistory(dir, []HistoryItem{})
}

func saveHistory(dir string, items []HistoryItem) error {
	if dir == "" {
		dir = AppDataDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}
	if items == nil {
		items = []HistoryItem{}
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}
	return writeFileAtomic(filepath.Join(dir, historyFileName), data)
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

