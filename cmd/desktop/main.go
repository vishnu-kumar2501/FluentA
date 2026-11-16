//go:build windows || linux

//go:generate go run gen_rsrc.go

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"grammar/desktop"
	"grammar/grammar"
)

func main() {
	var (
		baseURL       = flag.String("base-url", "", "Custom AI API base URL (overrides saved settings)")
		model         = flag.String("model", "", "Custom AI model name (overrides saved settings)")
		think         = flag.Bool("think", false, "Enable model thinking output when supported")
		hotkey        = flag.String("hotkey", "ctrl+alt+g", "Global hotkey to capture selection and open popup")
		modeFlag      = flag.String("mode", string(grammar.ModeFix), "Default mode: fix, enhance, formal, concise, explain")
		watch         = flag.Bool("watch", true, "Enable background UI Automation text watcher + floating badge")
		watchDebounce = flag.Duration("watch-debounce", 900*time.Millisecond, "Debounce wait after typing stops before background check")
		watchDebug    = flag.Bool("watch-debug", false, "Log UI Automation watcher element detection details")
	)
	flag.Parse()

	flagsSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagsSet[f.Name] = true })

	defaultMode, err := grammar.ParseMode(*modeFlag)
	if err != nil {
		log.Fatalf("invalid -mode: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dataDir := desktop.AppDataDir()
	settings, hasSettings, _ := desktop.LoadSettings(dataDir)
	if !hasSettings {
		settings = desktop.NormalizeSettings(desktop.Settings{
			AIProviderType: "cloud",
			AIBaseURL:      desktop.DefaultCloudBaseURL,
			AIModel:        desktop.DefaultCloudModel,
			AIApiKey:       desktop.ResolveAPIKey(""),
		})
	}

	if flagsSet["base-url"] && strings.TrimSpace(*baseURL) != "" {
		settings.AIBaseURL = strings.TrimSpace(*baseURL)
	}
	if flagsSet["model"] && strings.TrimSpace(*model) != "" {
		settings.AIModel = strings.TrimSpace(*model)
	}
	if flagsSet["mode"] {
		settings.DefaultMode = string(defaultMode)
	}
	if flagsSet["hotkey"] && strings.TrimSpace(*hotkey) != "" {
		settings.Hotkey = strings.TrimSpace(*hotkey)
	}
	if flagsSet["watch"] {
		v := *watch
		settings.WatchEnabled = &v
	}

	provider, err := desktop.BuildProvider(settings)
	if err != nil {
		log.Printf("provider build notice: %v, falling back to default cloud provider", err)
		provider, _ = desktop.BuildProvider(desktop.NormalizeSettings(desktop.Settings{
			AIProviderType: "cloud",
			AIBaseURL:      desktop.DefaultCloudBaseURL,
			AIModel:        desktop.DefaultCloudModel,
			AIApiKey:       desktop.ResolveAPIKey(""),
		}))
	}

	checker, err := grammar.NewChecker(provider, grammar.WithThinking(*think))
	if err != nil {
		log.Fatalf("initialization error: %v", err)
	}

	watchEnabled := true
	if settings.WatchEnabled != nil {
		watchEnabled = *settings.WatchEnabled
	}

	if err := desktop.Run(ctx, checker, desktop.Config{
		Hotkey:        settings.Hotkey,
		DefaultMode:   defaultMode,
		ModelName:     settings.AIModel,
		WatchEnabled:  watchEnabled,
		WatchDebounce: *watchDebounce,
		WatchDebug:    *watchDebug,
	}); err != nil {
		log.Fatalf("desktop app exited with error: %v", err)
	}
}
