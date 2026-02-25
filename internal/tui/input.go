package tui

import "unicode/utf8"

// pageSize is the default number of items fetched per API call.
const pageSize = 50

// editRune processes a keystroke for inline text editing.
// Handles backspace (rune-aware) and single printable characters.
// Returns the text unchanged for non-printable keys (enter, esc, etc.).
func editRune(text string, key string) string {
	switch key {
	case "backspace":
		if len(text) > 0 {
			runes := []rune(text)
			return string(runes[:len(runes)-1])
		}
		return text
	default:
		if utf8.RuneCountInString(key) == 1 {
			return text + key
		}
		return text
	}
}

// truncateToHeight limits output to maxLines newline-delimited lines.
// Returns the original string if it fits or maxLines is <= 0.
func truncateToHeight(s string, maxLines int) string {
	if maxLines <= 0 {
		return s
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
			if n >= maxLines {
				return s[:i+1]
			}
		}
	}
	return s
}
