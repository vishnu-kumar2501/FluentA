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

	"grammar/grammar"
)

// Run starts the FluentA desktop assistant on Linux.
func Run(ctx context.Context, checker *grammar.Checker, cfg Config) error {
	dataDir := AppDataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory %s: %w", dataDir, err)
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

	app := &linuxApp{
		checker:  checker,
		cfg:      cfg,
		dataDir:  dataDir,
		settings: settings,
		history:  history,
		sseChan:  make(chan hostEvent, 64),
	}

	return app.start(ctx)
}

type linuxApp struct {
	checker  *grammar.Checker
	cfg      Config
	dataDir  string
	settings Settings
	history  []HistoryItem

	mu      sync.Mutex
	sseChan chan hostEvent
	server  *http.Server
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

		// Send initial config snapshot
		initEvt := a.makeConfigEvent()
		data, _ := json.Marshal(initEvt)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		// Read active Linux selection on open
		if selText, err := ReadSelection(r.Context()); err == nil && strings.TrimSpace(selText) != "" {
			selEvt := hostEvent{
				Type:             "source",
				SourceText:       selText,
				CanReplaceTarget: true,
			}
			selData, _ := json.Marshal(selEvt)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", selData)
			flusher.Flush()
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
		Mode:           a.settings.DefaultMode,
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

	case "check":
		mode, _ := grammar.ParseMode(msg.Mode)
		res, err := a.checker.CheckModeStream(ctx, msg.Text, mode, nil)
		if err != nil {
			a.broadcast(hostEvent{Type: "error", Error: err.Error()})
			return
		}
		a.broadcast(hostEvent{
			Type:       "result",
			SourceText: msg.Text,
			OutputText: res.Message.Content,
			Mode:       string(mode),
		})

		// Add to history
		item := HistoryItem{
			ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
			Timestamp:  time.Now().Format(time.RFC3339),
			Mode:       string(mode),
			SourceText: msg.Text,
			OutputText: res.Message.Content,
		}
		a.mu.Lock()
		a.history, _ = AppendHistory(a.dataDir, item, 50)
		a.mu.Unlock()

	case "replace":
		// Replace selected text in active Linux window via clipboard & keystroke simulation
		if err := ReplaceSelectedText(ctx, msg.Revised); err != nil {
			log.Printf("replace text error: %v", err)
			a.broadcast(hostEvent{Type: "error", Error: err.Error()})
		}

	case "settings":
		if msg.Settings != nil {
			a.mu.Lock()
			a.settings = *msg.Settings
			_ = SaveSettings(a.dataDir, a.settings)
			a.mu.Unlock()
			a.broadcast(a.makeConfigEvent())
		}
	}
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
