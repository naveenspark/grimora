package tui

import "unicode/utf8"

// pageSize is the default number of items fetched per API call.
const pageSize = 50

// maxInputLen is the maximum number of runes allowed in chat and form inputs.
const maxInputLen = 2000

// editRune processes a keystroke for inline text editing.
// Handles backspace (rune-aware) and single printable characters.
// Returns the text unchanged for non-printable keys (enter, esc, etc.).
// Input is clamped to maxInputLen runes.
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
			if utf8.RuneCountInString(text) >= maxInputLen {
				return text
			}
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

// renderChatInput renders the shared inline text input used by Hall and Threads.
// It shows an animated name, cursor blink, and placeholder when empty.
func renderChatInput(login, input, placeholder string, focused bool, animFrame int) string {
	const timeIndent = "           " // 11 spaces — matches " " + 8-char timestamp + "  "

	sep := chatSepStyle.Render(" · ")
	namePart := renderAnimatedName(login, animFrame)
	if !focused {
		if input == "" {
			return timeIndent + namePart + sep + inputPlaceholderStyle.Render(placeholder)
		}
		return timeIndent + namePart + sep + dimStyle.Render(input)
	}
	cursor := " "
	if (animFrame/4)%2 == 0 {
		cursor = accentStyle.Render("█")
	}
	if input == "" {
		return timeIndent + namePart + sep + cursor
	}
	return timeIndent + namePart + sep + chatComposingStyle.Render(input) + cursor
}
