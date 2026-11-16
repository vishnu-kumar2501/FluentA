package desktop

import (
	"embed"
	"encoding/base64"
	"fmt"
	"strings"

	"grammar/grammar"
)

//go:embed ui/index.html ui/styles.css ui/app.js ui/fluenta-logo.png
var uiFS embed.FS

type hostEvent struct {
	Type             string                      `json:"type"`
	SourceText       string                      `json:"sourceText,omitempty"`
	OutputText       string                      `json:"outputText,omitempty"`
	Text             string                      `json:"text,omitempty"`
	Mode             string                      `json:"mode,omitempty"`
	DefaultMode      string                      `json:"defaultMode,omitempty"`
	ReplaceAllMode   bool                        `json:"replaceAllMode,omitempty"`
	CanReplaceTarget bool                        `json:"canReplaceTarget,omitempty"`
	HotkeyLabel      string                      `json:"hotkeyLabel,omitempty"`
	HotkeyAvailable  bool                        `json:"hotkeyAvailable,omitempty"`
	HotkeyConflict   bool                        `json:"hotkeyConflict,omitempty"`
	HotkeyNotice     string                      `json:"hotkeyNotice,omitempty"`
	ModelName        string                      `json:"modelName,omitempty"`
	Streaming        bool                        `json:"streaming,omitempty"`
	Error            string                      `json:"error,omitempty"`
	RequestID        string                      `json:"requestId,omitempty"`
	Explanations     []grammar.ChangeExplanation `json:"explanations,omitempty"`
	Insights         *grammar.WritingInsights    `json:"insights,omitempty"`
	WatchEnabled     bool                        `json:"watchEnabled"`
	Paused           bool                        `json:"paused"`
	Language         string                      `json:"language,omitempty"`
	Theme            string                      `json:"theme,omitempty"`
	ExcludedApps     []string                    `json:"excludedApps,omitempty"`
	History          []HistoryItem               `json:"history,omitempty"`
	ChunkInfo        string                      `json:"chunkInfo,omitempty"`
	AIConnected      bool                        `json:"aiConnected"`
	AIError          string                      `json:"aiError,omitempty"`
	AIProviderType   string                      `json:"aiProviderType,omitempty"`
	AIBaseURL        string                      `json:"aiBaseURL,omitempty"`
	AIModel          string                      `json:"aiModel,omitempty"`
	AIApiKey         string                      `json:"aiApiKey,omitempty"`
}

type uiMessage struct {
	Type           string               `json:"type"`
	Text           string               `json:"text,omitempty"`
	Mode           string               `json:"mode,omitempty"`
	Theme          string               `json:"theme,omitempty"`
	Edge           string               `json:"edge,omitempty"`
	RequestID      string               `json:"requestId,omitempty"`
	Original       string               `json:"original,omitempty"`
	Revised        string               `json:"revised,omitempty"`
	Changes        []grammar.DiffChange `json:"changes,omitempty"`
	Enabled        bool                 `json:"enabled,omitempty"`
	Language       string               `json:"language,omitempty"`
	Hotkey         string               `json:"hotkey,omitempty"`
	Settings       *Settings            `json:"settings,omitempty"`
	AIProviderType string               `json:"aiProviderType,omitempty"`
	AIBaseURL      string               `json:"aiBaseURL,omitempty"`
	AIModel        string               `json:"aiModel,omitempty"`
	AIApiKey       string               `json:"aiApiKey,omitempty"`
}

// buildPopupHTML assembles the standalone HTML document for WebView2 NavigateToString
// by inlining the embedded stylesheet and script into ui/index.html.
func buildPopupHTML() (string, error) {
	htmlBytes, err := uiFS.ReadFile("ui/index.html")
	if err != nil {
		return "", fmt.Errorf("read embedded index.html: %w", err)
	}
	cssBytes, err := uiFS.ReadFile("ui/styles.css")
	if err != nil {
		return "", fmt.Errorf("read embedded styles.css: %w", err)
	}
	jsBytes, err := uiFS.ReadFile("ui/app.js")
	if err != nil {
		return "", fmt.Errorf("read embedded app.js: %w", err)
	}
	logoBytes, err := uiFS.ReadFile("ui/fluenta-logo.png")
	if err != nil {
		return "", fmt.Errorf("read embedded fluenta-logo.png: %w", err)
	}

	const styleTag = `<link rel="stylesheet" href="styles.css">`
	const scriptTag = `<script src="app.js" defer></script>`

	doc := string(htmlBytes)
	if !strings.Contains(doc, styleTag) {
		return "", fmt.Errorf("embedded index.html missing stylesheet link marker %q", styleTag)
	}
	if !strings.Contains(doc, scriptTag) {
		return "", fmt.Errorf("embedded index.html missing script marker %q", scriptTag)
	}

	logoDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(logoBytes)
	safeCSS := strings.ReplaceAll(string(cssBytes), "</style>", `<\/style>`)
	safeJS := strings.ReplaceAll(string(jsBytes), "</script>", `<\/script>`)

	doc = strings.ReplaceAll(doc, `"fluenta-logo.png"`, `"`+logoDataURI+`"`)
	doc = strings.Replace(doc, styleTag, "<style>\n"+safeCSS+"\n</style>", 1)
	doc = strings.Replace(doc, scriptTag, "<script>\n"+safeJS+"\n</script>", 1)
	return doc, nil
}

