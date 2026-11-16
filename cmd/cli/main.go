package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"grammar/desktop"
	"grammar/grammar"
)

func main() {
	var (
		modeFlag    = flag.String("mode", string(grammar.ModeFix), "Proofreading mode: fix, enhance, formal, concise, explain")
		baseURL     = flag.String("base-url", "", "Custom AI base URL (default: FluentA Cloud Gateway)")
		model       = flag.String("model", "", "Custom AI model name")
		apiKey      = flag.String("api-key", "", "Custom API key (optional with Cloudflare Gateway)")
		jsonOut     = flag.Bool("json", false, "Output results as structured JSON")
		useSelect   = flag.Bool("selection", false, "Read text from the active Linux/X11/Wayland highlighted selection")
		replaceMode = flag.Bool("replace", false, "Replace the selected text in-place in the active application")
		timeoutSec  = flag.Int("timeout", 30, "Request timeout in seconds")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "FluentA CLI — Fast, Private AI Writing Assistant\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  fluenta [options] \"text to check\"\n")
		fmt.Fprintf(os.Stderr, "  echo \"some text\" | fluenta [options]\n")
		fmt.Fprintf(os.Stderr, "  fluenta --selection --replace\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()

	// 1. Determine input text
	var inputText string

	if *useSelect {
		// Read from active selection
		sel, err := desktop.ReadSelection(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading selection: %v\n", err)
			os.Exit(1)
		}
		inputText = strings.TrimSpace(sel)
		if inputText == "" {
			fmt.Fprintln(os.Stderr, "No text is currently selected.")
			os.Exit(0)
		}
	} else if flag.NArg() > 0 {
		inputText = strings.Join(flag.Args(), " ")
	} else {
		// Check if piped from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
			inputText = string(bytes)
		}
	}

	inputText = strings.TrimSpace(inputText)
	if inputText == "" {
		flag.Usage()
		os.Exit(1)
	}

	// 2. Parse mode
	mode, err := grammar.ParseMode(*modeFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid mode %q: %v\n", *modeFlag, err)
		os.Exit(1)
	}

	// 3. Initialize Provider & Checker
	dataDir := desktop.AppDataDir()
	settings, _, _ := desktop.LoadSettings(dataDir)
	settings = desktop.NormalizeSettings(settings)

	if strings.TrimSpace(*baseURL) != "" {
		settings.AIBaseURL = strings.TrimSpace(*baseURL)
	}
	if strings.TrimSpace(*model) != "" {
		settings.AIModel = strings.TrimSpace(*model)
	}
	if strings.TrimSpace(*apiKey) != "" {
		settings.AIApiKey = strings.TrimSpace(*apiKey)
	}

	provider, err := desktop.BuildProvider(settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize AI provider: %v\n", err)
		os.Exit(1)
	}

	checker, err := grammar.NewChecker(provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize grammar checker: %v\n", err)
		os.Exit(1)
	}

	// 4. Execute check
	resp, err := checker.CheckModeStream(ctx, inputText, mode, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Proofreading error: %v\n", err)
		os.Exit(1)
	}

	corrected := strings.TrimSpace(resp.Message.Content)

	// 5. If replace requested, replace in active Linux window
	if *replaceMode {
		if err := desktop.ReplaceSelectedText(ctx, corrected); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to replace in-place: %v\n", err)
		}
	}

	// 6. Output result
	if *jsonOut {
		outData := map[string]interface{}{
			"original":  inputText,
			"corrected": corrected,
			"mode":      string(mode),
			"model":     settings.AIModel,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(outData)
	} else {
		fmt.Println(corrected)
	}
}
