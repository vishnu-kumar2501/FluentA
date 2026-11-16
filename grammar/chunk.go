package grammar

import (
	"strings"
	"unicode"
)

// DefaultMaxChunkChars is the character threshold above which long inputs are
// split cleanly along paragraph or sentence boundaries for sequential checking.
const DefaultMaxChunkChars = 2800

// SplitTextChunks splits text into pieces no longer than maxChars (when possible),
// preferring double-newline paragraph boundaries, then sentence endings, then whitespace.
// Concatenating all returned chunks reproduces the original text identically.
func SplitTextChunks(text string, maxChars int) []string {
	if text == "" {
		return nil
	}
	if maxChars <= 0 {
		maxChars = DefaultMaxChunkChars
	}
	if len(text) <= maxChars {
		return []string{text}
	}

	var chunks []string
	start := 0
	n := len(text)

	for start < n {
		if n-start <= maxChars {
			chunks = append(chunks, text[start:])
			break
		}

		limit := start + maxChars
		cut := findBestSplitIndex(text, start, limit)
		if cut <= start {
			cut = limit
		}
		chunks = append(chunks, text[start:cut])
		start = cut
	}
	return chunks
}

func findBestSplitIndex(text string, start, limit int) int {
	window := text[start:limit]

	// 1. Prefer paragraph boundary ("\n\n") near the end of window, skipping index 0.
	if idx := strings.LastIndex(window, "\n\n"); idx > 0 {
		return start + idx + 2
	}

	// 2. Next prefer single line break.
	if idx := strings.LastIndexByte(window, '\n'); idx > 0 {
		return start + idx + 1
	}

	// 3. Next prefer sentence boundary (. ! ? followed by whitespace).
	for i := len(window) - 1; i > 0; i-- {
		ch := window[i-1]
		if (ch == '.' || ch == '!' || ch == '?') && unicode.IsSpace(rune(window[i])) {
			return start + i + 1
		}
	}

	// 4. Next prefer any whitespace boundary.
	for i := len(window) - 1; i > 0; i-- {
		if unicode.IsSpace(rune(window[i])) {
			return start + i + 1
		}
	}

	// 5. Fallback: avoid splitting multi-byte UTF-8 runes mid-sequence.
	cut := limit
	for cut > start && (text[cut]&0xC0) == 0x80 {
		cut--
	}
	return cut
}

