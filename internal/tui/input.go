package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// namedKeys is the set of Bubbletea key names that are multi-character
// strings composed of printable ASCII. These must not be treated as paste.
var namedKeys = map[string]bool{
	"enter": true, "return": true, "tab": true, "esc": true, "escape": true,
	"space": true, "up": true, "down": true, "left": true, "right": true,
	"home": true, "end": true, "pgup": true, "pgdown": true,
	"delete": true, "insert": true,
	"f1": true, "f2": true, "f3": true, "f4": true, "f5": true,
	"f6": true, "f7": true, "f8": true, "f9": true, "f10": true,
	"f11": true, "f12": true, "f13": true, "f14": true, "f15": true,
	"f16": true, "f17": true, "f18": true, "f19": true, "f20": true,
}

func isNamedKey(key string) bool {
	if namedKeys[key] {
		return true
	}
	if strings.HasPrefix(key, "ctrl+") || strings.HasPrefix(key, "alt+") || strings.HasPrefix(key, "shift+") {
		return true
	}
	return false
}

// pageSize is the default number of items fetched per API call.
const pageSize = 50

// maxInputLen is the maximum number of runes allowed in chat and form inputs.
const maxInputLen = 2000

// editRune processes a keystroke for inline text editing.
// Handles backspace (rune-aware), single printable characters, and multi-rune
// paste strings. Returns the text unchanged for non-printable keys (enter, esc, etc.).
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
		keyLen := utf8.RuneCountInString(key)
		if keyLen < 1 || isNamedKey(key) {
			return text
		}
		textLen := utf8.RuneCountInString(text)
		if textLen >= maxInputLen {
			return text
		}
		// Clamp paste to remaining capacity.
		if keyLen > 1 {
			remaining := maxInputLen - textLen
			if keyLen > remaining {
				runes := []rune(key)
				key = string(runes[:remaining])
			}
		}
		return text + key
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

// inputPrefixWidth returns the visual width of the chat input prefix
// (timeIndent + name + separator) for a given login.
func inputPrefixWidth(login string) int {
	const timeIndent = "           " // 11 spaces
	sep := chatSepStyle.Render(" · ")
	namePart := renderAnimatedName(login, 0) // frame irrelevant for width
	return lipgloss.Width(timeIndent + namePart + sep)
}

// wrapInputLines takes a single logical line and wraps it to fit bodyWidth,
// using lipgloss word-wrap first, then hardWrap for long tokens.
// Returns the resulting visual lines.
func wrapInputLines(line string, bodyWidth int) []string {
	if bodyWidth <= 0 {
		return []string{line}
	}
	wrapped := lipgloss.NewStyle().Width(bodyWidth).Render(line)
	wrapped = stripTrailingSpaces(wrapped)
	wrapped = hardWrap(wrapped, bodyWidth)
	return strings.Split(wrapped, "\n")
}

// countInputVisualLines counts how many visual lines the input produces
// after wrapping to bodyWidth. Used by Hall and Threads chrome calculations.
func countInputVisualLines(input string, bodyWidth int) int {
	if bodyWidth <= 0 {
		return 1 + strings.Count(input, "\n")
	}
	count := 0
	for _, line := range strings.Split(input, "\n") {
		vlines := wrapInputLines(line, bodyWidth)
		count += len(vlines)
	}
	if count < 1 {
		count = 1
	}
	return count
}

// renderChatInput renders the shared inline text input used by Hall and Threads.
// It shows an animated name, cursor blink, and placeholder when empty.
// Supports multiline input: first line gets the name prefix, continuation lines
// are indented to align with the message body. Long lines are word-wrapped
// and hard-wrapped to fit within the given terminal width.
func renderChatInput(login, input, placeholder string, focused bool, animFrame int, width int) string {
	const timeIndent = "           " // 11 spaces — matches " " + 8-char timestamp + "  "

	sep := chatSepStyle.Render(" · ")
	namePart := renderAnimatedName(login, animFrame)
	if !focused {
		if input == "" {
			return timeIndent + namePart + sep + inputPlaceholderStyle.Render(placeholder)
		}
		// Show first line only when unfocused (multiline collapses)
		firstLine := input
		if idx := strings.IndexByte(input, '\n'); idx >= 0 {
			firstLine = input[:idx] + "…"
		}
		return timeIndent + namePart + sep + dimStyle.Render(firstLine)
	}
	cursor := " "
	if (animFrame/4)%2 == 0 {
		cursor = accentStyle.Render("█")
	}
	if input == "" {
		return timeIndent + namePart + sep + cursor
	}

	prefix := timeIndent + namePart + sep
	prefixWidth := lipgloss.Width(prefix)
	contIndent := strings.Repeat(" ", prefixWidth)
	bodyWidth := width - prefixWidth - 1 // -1 reserves a column for the cursor
	if bodyWidth < 10 {
		bodyWidth = 10
	}

	// Split by explicit newlines, then wrap each logical line.
	logicalLines := strings.Split(input, "\n")
	var b strings.Builder
	first := true
	for _, logLine := range logicalLines {
		vlines := wrapInputLines(logLine, bodyWidth)
		for _, vl := range vlines {
			if first {
				b.WriteString(prefix + chatComposingStyle.Render(vl))
				first = false
			} else {
				b.WriteByte('\n')
				b.WriteString(contIndent + chatComposingStyle.Render(vl))
			}
		}
	}
	b.WriteString(cursor)
	return b.String()
}
