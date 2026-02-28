package tui

import (
	"strings"
	"testing"
)

func TestEditRuneAddCharacters(t *testing.T) {
	tests := []struct {
		name  string
		start string
		key   string
		want  string
	}{
		{"append to empty", "", "a", "a"},
		{"append letter", "hel", "l", "hell"},
		{"append digit", "abc", "1", "abc1"},
		{"append space", "hello", " ", "hello "},
		{"append special", "abc", "!", "abc!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := editRune(tc.start, tc.key)
			if got != tc.want {
				t.Errorf("editRune(%q, %q) = %q, want %q", tc.start, tc.key, got, tc.want)
			}
		})
	}
}

func TestEditRuneBackspace(t *testing.T) {
	tests := []struct {
		name  string
		start string
		want  string
	}{
		{"backspace on single char", "a", ""},
		{"backspace on longer string", "hello", "hell"},
		{"backspace on empty does nothing", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := editRune(tc.start, "backspace")
			if got != tc.want {
				t.Errorf("editRune(%q, 'backspace') = %q, want %q", tc.start, got, tc.want)
			}
		})
	}
}

func TestEditRuneBackspaceMultibyte(t *testing.T) {
	// Backspace should remove a full rune, not just one byte.
	// "héllo" ends with 'o' so backspace removes 'o', giving "héll".
	got := editRune("héllo", "backspace")
	if got != "héll" {
		t.Errorf("editRune(multi-byte, backspace) = %q, want %q", got, "héll")
	}

	// Backspace on a string ending with a multi-byte rune removes that rune cleanly.
	// "hellé" — backspace removes 'é' (2 bytes), leaving "hell".
	got2 := editRune("hellé", "backspace")
	if got2 != "hell" {
		t.Errorf("editRune ending with multi-byte rune: = %q, want %q", got2, "hell")
	}
}

func TestEditRuneIgnoresNonPrintableKeys(t *testing.T) {
	nonPrintable := []string{
		"enter",
		"esc",
		"up",
		"down",
		"left",
		"right",
		"ctrl+c",
		"ctrl+s",
		"tab",
		"shift+tab",
		"f1",
		"pgup",
		"pgdown",
		"home",
		"end",
	}

	original := "hello"
	for _, key := range nonPrintable {
		t.Run(key, func(t *testing.T) {
			got := editRune(original, key)
			if got != original {
				t.Errorf("editRune(%q, %q) = %q, want unchanged %q", original, key, got, original)
			}
		})
	}
}

func TestTruncStr(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"under limit", "hello", 10, "hello"},
		{"at limit", "hello", 5, "hello"},
		{"over limit", "hello world", 5, "hell\u2026"},
		{"empty string", "", 5, ""},
		{"single char over", "ab", 1, "\u2026"},
		{"emoji", "\U0001f600\U0001f601\U0001f602", 2, "\U0001f600\u2026"},
		{"CJK chars", "\u4f60\u597d\u4e16\u754c", 3, "\u4f60\u597d\u2026"},
		{"multi-byte at boundary", "caf\u00e9s are nice", 5, "caf\u00e9\u2026"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncStr(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncStr(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestEditRuneBackspaceEmoji(t *testing.T) {
	// Backspace on a string ending with a 4-byte emoji should remove the whole emoji.
	got := editRune("hello\U0001f600", "backspace")
	if got != "hello" {
		t.Errorf("editRune with trailing emoji: got %q, want %q", got, "hello")
	}
}

func TestEditRunePaste(t *testing.T) {
	tests := []struct {
		name  string
		start string
		key   string
		want  string
	}{
		{"paste into empty", "", "hello world", "hello world"},
		{"paste appends", "hi ", "there", "hi there"},
		{"paste with special chars", "", "curl -fsSL https://grimora.ai/install.sh | sh", "curl -fsSL https://grimora.ai/install.sh | sh"},
		{"paste clamped at limit", strings.Repeat("a", maxInputLen-3), "abcdef", strings.Repeat("a", maxInputLen-3) + "abc"},
		{"paste rejected at limit", strings.Repeat("a", maxInputLen), "hello", strings.Repeat("a", maxInputLen)},
		{"named keys still ignored", "hello", "enter", "hello"},
		{"ctrl combos still ignored", "hello", "ctrl+c", "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := editRune(tc.start, tc.key)
			if got != tc.want {
				t.Errorf("editRune(%q, %q) = %q, want %q", tc.start, tc.key, got, tc.want)
			}
		})
	}
}

func TestEditRuneMaxInputLen(t *testing.T) {
	atLimit := strings.Repeat("a", maxInputLen)         // 2000 ASCII runes
	belowLimit := strings.Repeat("a", maxInputLen-1)    // 1999 ASCII runes
	cjkAtLimit := strings.Repeat("\u4f60", maxInputLen) // 2000 CJK runes (6000 bytes)

	tests := []struct {
		name string
		text string
		key  string
		want string
	}{
		{"at limit rejects new char", atLimit, "b", atLimit},
		{"below limit accepts new char", belowLimit, "b", belowLimit + "b"},
		{"at limit backspace still works", atLimit, "backspace", atLimit[:len(atLimit)-1]},
		{"at limit non-printable ignored", atLimit, "enter", atLimit},
		{"CJK at limit rejects new rune", cjkAtLimit, "\u597d", cjkAtLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := editRune(tt.text, tt.key)
			if got != tt.want {
				t.Errorf("editRune(..., %q): len(got)=%d runes, len(want)=%d runes, changed=%v",
					tt.key, len([]rune(got)), len([]rune(tt.want)), got != tt.text)
			}
		})
	}
}

func TestTruncateToHeightLimitsLines(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5\n"
	result := truncateToHeight(input, 3)

	lines := strings.Count(result, "\n")
	if lines > 3 {
		t.Errorf("truncateToHeight(5 lines, 3) produced %d newlines, want <= 3", lines)
	}
	if !strings.Contains(result, "line1") {
		t.Errorf("truncateToHeight result missing first line: %q", result)
	}
	if strings.Contains(result, "line4") {
		t.Errorf("truncateToHeight result should not contain line4: %q", result)
	}
}

func TestTruncateToHeightReturnsFullStringWhenWithinLimit(t *testing.T) {
	input := "line1\nline2\nline3\n"
	result := truncateToHeight(input, 10)
	if result != input {
		t.Errorf("truncateToHeight with maxLines > linecount: got %q, want %q", result, input)
	}
}

func TestTruncateToHeightZeroMaxReturnsAll(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5\n"
	result := truncateToHeight(input, 0)
	if result != input {
		t.Errorf("truncateToHeight with maxLines=0 should return input unchanged, got %q", result)
	}
}

func TestTruncateToHeightNegativeMaxReturnsAll(t *testing.T) {
	input := "line1\nline2\n"
	result := truncateToHeight(input, -1)
	if result != input {
		t.Errorf("truncateToHeight with maxLines=-1 should return input unchanged, got %q", result)
	}
}

func TestTruncateToHeightExactLimit(t *testing.T) {
	input := "line1\nline2\nline3\n"
	result := truncateToHeight(input, 3)
	// Should include all lines when count equals limit
	if !strings.Contains(result, "line1") || !strings.Contains(result, "line2") || !strings.Contains(result, "line3") {
		t.Errorf("truncateToHeight at exact limit dropped lines: %q", result)
	}
}

func TestEditRuneShiftEnterIgnored(t *testing.T) {
	// editRune should NOT insert a newline for shift+enter — that's handled
	// in hall.go's updateInput, not editRune. editRune sees it as a named key.
	keys := []string{"shift+enter", "alt+enter"}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			got := editRune("hello", key)
			if got != "hello" {
				t.Errorf("editRune(%q, %q) = %q, want unchanged", "hello", key, got)
			}
		})
	}
}

func TestRenderChatInputMultiline(t *testing.T) {
	// Multiline input should render continuation lines indented.
	result := renderChatInput("testuser", "line1\nline2", "placeholder", true, 0, 80)
	if !strings.Contains(result, "line1") || !strings.Contains(result, "line2") {
		t.Errorf("multiline input missing lines: %q", result)
	}
	// Should contain a newline (multi-line rendering).
	if !strings.Contains(result, "\n") {
		t.Errorf("multiline input should produce newlines in output: %q", result)
	}
}

func TestRenderChatInputUnfocusedMultilineCollapse(t *testing.T) {
	// When unfocused, multiline input shows first line with ellipsis.
	result := renderChatInput("testuser", "line1\nline2\nline3", "placeholder", false, 0, 80)
	if !strings.Contains(result, "line1") {
		t.Errorf("unfocused multiline should show first line: %q", result)
	}
	if strings.Contains(result, "line2") {
		t.Errorf("unfocused multiline should NOT show continuation lines: %q", result)
	}
}

func TestRenderChatInputWrapsLongLine(t *testing.T) {
	// A single long line should wrap to multiple visual lines.
	longInput := strings.Repeat("abcdefghij ", 10) // ~110 chars
	result := renderChatInput("testuser", longInput, "placeholder", true, 0, 60)
	newlines := strings.Count(result, "\n")
	if newlines < 1 {
		t.Errorf("long input should wrap to multiple lines, got 0 newlines: %q", result)
	}
}

func TestRenderChatInputPreservesTrailingSpace(t *testing.T) {
	// Trailing spaces must be visible so the cursor sits after them.
	// Use animFrame=1 so cursor is a plain space (not █) for easier assertion.
	result := renderChatInput("testuser", "hello ", "placeholder", true, 2, 80)
	// The rendered output should contain "hello " (with space) before the cursor.
	if !strings.Contains(result, "hello ") {
		t.Errorf("trailing space lost in render: %q", result)
	}
	// Multiple trailing spaces
	result2 := renderChatInput("testuser", "hello   ", "placeholder", true, 2, 80)
	if !strings.Contains(result2, "hello   ") {
		t.Errorf("multiple trailing spaces lost in render: %q", result2)
	}
}

func TestCountInputVisualLines(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		bodyWidth int
		wantMin   int
		wantMax   int
	}{
		{"empty", "", 60, 1, 1},
		{"short single line", "hello", 60, 1, 1},
		{"two explicit newlines", "line1\nline2", 60, 2, 2},
		{"long wraps", strings.Repeat("x", 120), 60, 2, 3},
		{"mixed newlines and wrapping", "short\n" + strings.Repeat("y", 120), 60, 3, 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := countInputVisualLines(tc.input, tc.bodyWidth)
			if got < tc.wantMin || got > tc.wantMax {
				t.Errorf("countInputVisualLines(%q, %d) = %d, want [%d, %d]",
					tc.input, tc.bodyWidth, got, tc.wantMin, tc.wantMax)
			}
		})
	}
}
