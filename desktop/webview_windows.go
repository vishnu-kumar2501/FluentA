//go:build windows

package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jchv/go-webview2/pkg/edge"
)

type webViewHost struct {
	chromium  *edge.Chromium
	ready     bool
	pending   []hostEvent
	onMessage func(uiMessage)
}

func newWebViewHost(hwnd uintptr, onMessage func(uiMessage)) (*webViewHost, error) {
	htmlDoc, err := buildPopupHTML()
	if err != nil {
		return nil, err
	}

	dataPath := webViewDataPath()
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		log.Printf("webview cache dir creation warning (%s): %v", dataPath, err)
	}

	chromium := edge.NewChromium()
	chromium.DataPath = dataPath

	host := &webViewHost{
		chromium:  chromium,
		onMessage: onMessage,
	}

	chromium.MessageCallback = func(raw string) {
		var msg uiMessage
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			log.Printf("webview message decode error: %v", err)
			return
		}
		if msg.Type == "ready" {
			host.markReady()
		}
		if host.onMessage != nil {
			host.onMessage(msg)
		}
	}

	chromium.NavigationCompletedCallback = func(_ *edge.ICoreWebView2, _ *edge.ICoreWebView2NavigationCompletedEventArgs) {
		host.resize()
	}

	chromium.AcceleratorKeyCallback = func(vkey uint) bool {
		if vkey == vkEscape {
			postMessage(hwnd, wmClose, 0, 0)
			return true
		}
		return false
	}

	if !chromium.Embed(hwnd) {
		return nil, errors.New("WebView2 runtime initialization failed; ensure Microsoft Edge WebView2 Runtime is installed")
	}

	if settings, err := chromium.GetSettings(); err == nil && settings != nil {
		_ = settings.PutIsStatusBarEnabled(false)
		_ = settings.PutIsZoomControlEnabled(false)
		_ = settings.PutIsBuiltInErrorPageEnabled(false)
	}

	chromium.Resize()
	chromium.NavigateToString(htmlDoc)
	return host, nil
}

func webViewDataPath() string {
	if custom := os.Getenv("GRAMMAR_WEBVIEW_DATA_DIR"); custom != "" {
		return custom
	}
	if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		return filepath.Join(cacheDir, "FluentA", "WebView2")
	}
	return filepath.Join(os.TempDir(), "FluentA", "WebView2")
}

func (w *webViewHost) markReady() {
	if w == nil {
		return
	}
	w.ready = true
	if len(w.pending) == 0 {
		return
	}
	queued := w.pending
	w.pending = nil
	for _, evt := range queued {
		w.send(evt)
	}
}

func (w *webViewHost) queueEvent(evt hostEvent) {
	// Replace earlier full-open payloads so stale initial states do not replay.
	if evt.Type == "open" {
		filtered := w.pending[:0]
		for _, p := range w.pending {
			if p.Type == "config" {
				filtered = append(filtered, p)
			}
		}
		w.pending = append(filtered, evt)
		return
	}
	w.pending = append(w.pending, evt)
}

func (w *webViewHost) send(evt hostEvent) {
	if w == nil || w.chromium == nil {
		return
	}
	if !w.ready {
		w.queueEvent(evt)
		return
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}
	script := fmt.Sprintf("window.__grammarApp && window.__grammarApp.onHostEvent(%s);", string(payload))
	w.chromium.Eval(script)
}

func (w *webViewHost) resize() {
	if w != nil && w.chromium != nil {
		w.chromium.Resize()
	}
}

func (w *webViewHost) notifyMoved() {
	if w != nil && w.chromium != nil {
		_ = w.chromium.NotifyParentWindowPositionChanged()
	}
}

func (w *webViewHost) show() {
	if w != nil && w.chromium != nil {
		_ = w.chromium.Show()
	}
}

func (w *webViewHost) hide() {
	if w != nil && w.chromium != nil {
		_ = w.chromium.Hide()
	}
}

func (w *webViewHost) focus() {
	if w != nil && w.chromium != nil {
		w.chromium.Focus()
	}
}
