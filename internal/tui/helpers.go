package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// formatTime renders a relative timestamp for stream/workshop displays.
func formatTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// truncStr truncates a string to maxLen runes, appending an ellipsis if needed.
func truncStr(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen-1]) + "\u2026"
}

// cleanTitle strips markdown headers and collapses whitespace from a spell title
// so stream entries show meaningful content instead of "# Header Name".
func cleanTitle(raw string) string {
	// Replace newlines with spaces first
	s := strings.ReplaceAll(raw, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	// Strip leading markdown header markers (# ## ### etc.)
	for strings.HasPrefix(s, "#") {
		s = strings.TrimLeft(s, "#")
		s = strings.TrimLeft(s, " ")
	}

	// Collapse runs of whitespace
	parts := strings.Fields(s)
	s = strings.Join(parts, " ")

	return strings.TrimSpace(s)
}
