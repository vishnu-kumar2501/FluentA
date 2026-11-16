//go:build linux

package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"grammar/ai"
	"grammar/grammar"
)

// Run starts the FluentA desktop assistant on Linux.
func Run(ctx context.Context, checker *grammar.Checker, cfg Config) error {
	dataDir := AppDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory %s: %w", dataDir, err)
	}

	closeLog, logErr := setupFileLogger(dataDir)
	if logErr == nil && closeLog != nil {
		defer closeLog()
	} else if logErr != nil {
		log.Printf("file logger warning: %v", logErr)
	}

	// Single instance enforcement
	lockName := instanceMutexName(dataDir)
	lock, err := acquireSingleInstance(lockName)
	if err != nil {
		return err
	}
	defer lock.Release()

	// Load settings & history
	settings, _, err := LoadSettings(dataDir)
	if err != nil {
		settings = NormalizeSettings(Settings{
			AIProviderType: "cloud",
			AIBaseURL:      DefaultCloudBaseURL,
			AIModel:        DefaultCloudModel,
			AIApiKey:       ResolveAPIKey(""),
		})
	}
	history, _ := LoadHistory(dataDir)

	defMode := cfg.DefaultMode
	if defMode == "" {
		defMode = grammar.ModeFix
	}

	app := &linuxApp{
		checker:     checker,
		cfg:         cfg,
		dataDir:     dataDir,
		settings:    settings,
		history:     history,
		currentMode: defMode,
		sseChan:     make(chan hostEvent, 128),
	}

	return app.start(ctx)
}

type linuxApp struct {
	checker     *grammar.Checker
	cfg         Config
	dataDir     string
	settings    Settings
	history     []HistoryItem
	currentMode grammar.Mode

	mu           sync.Mutex
	activeCancel context.CancelFunc
	sseChan      chan hostEvent
	server       *http.Server
}

func (a *linuxApp) broadcast(evt hostEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case a.sseChan <- evt:
	default:
		// Drop if buffer full
	}
}

func (a *linuxApp) start(ctx context.Context) error {
	// 1. Bind to a random free loopback port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("bind local loopback listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	mux := http.NewServeMux()

	// Serve main HTML app
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html, err := buildPopupHTML()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})

	// Server-Sent Events (SSE) stream for Go -> UI hostEvents
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// 1. Send initial config snapshot
		initEvt := a.makeConfigEvent()
		data, _ := json.Marshal(initEvt)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// 2. Read active Linux primary selection on open and populate editor
		selText, _ := ReadSelection(r.Context())
		trimmedSel := strings.TrimSpace(selText)

		openEvt := hostEvent{
			Type:             "open",
			SourceText:       trimmedSel,
			CanReplaceTarget: trimmedSel != "",
			Mode:             string(a.currentMode),
			ModelName:        a.settings.AIModel,
		}
		openData, _ := json.Marshal(openEvt)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", openData)
		flusher.Flush()

		// If text was selected, automatically trigger check
		if trimmedSel != "" {
			go a.executeCheck(context.Background(), trimmedSel, a.currentMode)
		}

		clientGone := r.Context().Done()
		for {
			select {
			case <-clientGone:
				return
			case evt := <-a.sseChan:
				evtData, _ := json.Marshal(evt)
				_, _ = fmt.Fprintf(w, "data: %s\n\n", evtData)
				flusher.Flush()
			}
		}
	})

	// JSON message endpoint for UI -> Go uiMessages
	mux.HandleFunc("/api/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var msg uiMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		go a.handleUIMessage(context.Background(), msg)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	a.server = &http.Server{Handler: mux}

	// 2. Launch webview / app window
	go func() {
		time.Sleep(100 * time.Millisecond)
		a.launchWindow(url)
	}()

	// 3. Wait for context cancellation or server error
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- a.server.Serve(listener)
	}()

	log.Printf("FluentA Linux Desktop running on %s", url)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = a.server.Shutdown(shutdownCtx)
		return nil
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

func (a *linuxApp) makeConfigEvent() hostEvent {
	watchVal := false
	if a.settings.WatchEnabled != nil {
		watchVal = *a.settings.WatchEnabled
	}

	return hostEvent{
		Type:           "config",
		HotkeyLabel:    a.settings.Hotkey,
		ModelName:      a.settings.AIModel,
		Mode:           string(a.currentMode),
		DefaultMode:    a.settings.DefaultMode,
		WatchEnabled:   watchVal,
		Paused:         a.settings.Paused,
		Language:       a.settings.Language,
		Theme:          a.settings.Theme,
		ExcludedApps:   a.settings.ExcludedApps,
		History:        a.history,
		AIConnected:    true,
		AIProviderType: a.settings.AIProviderType,
		AIBaseURL:      a.settings.AIBaseURL,
		AIModel:        a.settings.AIModel,
	}
}

func (a *linuxApp) handleUIMessage(ctx context.Context, msg uiMessage) {
	switch msg.Type {
	case "ready":
		a.broadcast(a.makeConfigEvent())

	case "mode":
		if m, err := grammar.ParseMode(msg.Mode); err == nil {
			a.mu.Lock()
			a.currentMode = m
			a.mu.Unlock()
			a.broadcast(hostEvent{Type: "mode", Mode: string(m)})
		}

	case "recheck", "check":
		targetMode := a.currentMode
		if msg.Mode != "" {
			if m, err := grammar.ParseMode(msg.Mode); err == nil {
				targetMode = m
				a.mu.Lock()
				a.currentMode = m
				a.mu.Unlock()
			}
		}
		text := strings.TrimSpace(msg.Text)
		if text == "" {
			a.broadcast(hostEvent{Type: "error", Error: "Input text is empty."})
			return
		}
		a.executeCheck(ctx, text, targetMode)

	case "stop":
		a.mu.Lock()
		if a.activeCancel != nil {
			a.activeCancel()
			a.activeCancel = nil
		}
		a.mu.Unlock()

	case "replace":
		text := msg.Text
		if text == "" {
			text = msg.Revised
		}
		if err := ReplaceSelectedText(ctx, text); err != nil {
			log.Printf("replace text error: %v", err)
			a.broadcast(hostEvent{Type: "error", Error: err.Error()})
		}

	case "copy":
		text := msg.Text
		if text == "" {
			text = msg.Revised
		}
		_ = WriteClipboard(ctx, text)

	case "clearHistory":
		_ = ClearHistory(a.dataDir)
		a.mu.Lock()
		a.history = nil
		a.mu.Unlock()
		a.broadcast(hostEvent{
			Type:    "history",
			History: []HistoryItem{},
		})

	case "explainChanges":
		go func() {
			explanations, err := a.checker.ExplainChanges(ctx, msg.Original, msg.Revised, msg.Changes)
			if err != nil {
				log.Printf("explainChanges error: %v", err)
				return
			}
			a.broadcast(hostEvent{
				Type:         "explanations",
				RequestID:    msg.RequestID,
				Explanations: explanations,
			})
		}()

	case "analyzeWriting":
		go func() {
			insights, err := a.checker.AnalyzeWriting(ctx, msg.Text, msg.Language)
			if err != nil {
				log.Printf("analyzeWriting error: %v", err)
				return
			}
			a.broadcast(hostEvent{
				Type:      "insights",
				RequestID: msg.RequestID,
				Insights:  &insights,
			})
		}()

	case "saveSettings", "settings":
		if msg.Settings != nil {
			a.mu.Lock()
			a.settings = *msg.Settings
			_ = SaveSettings(a.dataDir, a.settings)
			a.mu.Unlock()
			a.broadcast(a.makeConfigEvent())
		}
	}
}

func (a *linuxApp) executeCheck(ctx context.Context, text string, mode grammar.Mode) {
	a.mu.Lock()
	if a.activeCancel != nil {
		a.activeCancel()
	}
	inferCtx, cancel := context.WithCancel(ctx)
	a.activeCancel = cancel
	a.mu.Unlock()

	// Notify UI streaming has started
	a.broadcast(hostEvent{
		Type:      "start",
		Mode:      string(mode),
		Streaming: true,
	})

	go func() {
		defer cancel()
		var fullStreamed strings.Builder

		res, err := a.checker.CheckModeStream(inferCtx, text, mode, func(chunk ai.StreamChunk) error {
			if chunk.Content != "" {
				fullStreamed.WriteString(chunk.Content)
				a.broadcast(hostEvent{
					Type: "chunk",
					Text: chunk.Content,
				})
			}
			return nil
		})

		if err != nil {
			if inferCtx.Err() != context.Canceled {
				a.broadcast(hostEvent{
					Type:  "error",
					Error: err.Error(),
				})
			}
			return
		}

		finalOutput := res.Message.Content
		if finalOutput == "" {
			finalOutput = fullStreamed.String()
		}
		finalOutput = grammar.CleanOutput(finalOutput)

		// Notify UI completion
		a.broadcast(hostEvent{
			Type:       "done",
			Text:       finalOutput,
			OutputText: finalOutput,
			Mode:       string(mode),
		})

		// Add to history
		item := HistoryItem{
			ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
			Timestamp:  time.Now().Format(time.RFC3339),
			Mode:       string(mode),
			SourceText: text,
			OutputText: finalOutput,
		}
		a.mu.Lock()
		updated, _ := AppendHistory(a.dataDir, item, defaultHistoryMaxItems)
		a.history = updated
		a.mu.Unlock()

		a.broadcast(hostEvent{
			Type:    "history",
			History: updated,
		})
	}()
}

// launchWindow opens the app in a standalone app window using the best available Linux browser
func (a *linuxApp) launchWindow(url string) {
	candidates := []struct {
		bin  string
		args []string
	}{
		// Chromium-based browsers support --app which removes browser chrome/tabs
		{"google-chrome", []string{"--app=" + url, "--window-size=940,680", "--class=FluentA"}},
		{"chromium", []string{"--app=" + url, "--window-size=940,680", "--class=FluentA"}},
		{"chromium-browser", []string{"--app=" + url, "--window-size=940,680", "--class=FluentA"}},
		{"brave-browser", []string{"--app=" + url, "--window-size=940,680", "--class=FluentA"}},
		{"epiphany", []string{"--application-mode", url}},
		// Standard browser fallback
		{"xdg-open", []string{url}},
	}

	for _, c := range candidates {
		if isCommandAvailable(c.bin) {
			cmd := exec.Command(c.bin, c.args...)
			if err := cmd.Start(); err == nil {
				return
			}
		}
	}
}
