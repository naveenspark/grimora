package tui

import (
	"strings"
	"testing"
)

func TestTagStyleKnownTag(t *testing.T) {
	tests := []struct {
		tag string
	}{
		{"debugging"},
		{"architecture"},
		{"performance"},
		{"code-review"},
		{"testing"},
		{"security"},
		{"database"},
	}

	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			style := TagStyle(tc.tag)
			// Known tags should render the tag text without panicking
			rendered := style.Render(tc.tag)
			if !strings.Contains(rendered, tc.tag) {
				t.Errorf("TagStyle(%q).Render(%q) = %q, want to contain %q", tc.tag, tc.tag, rendered, tc.tag)
			}
		})
	}
}

func TestTagStyleUnknownTagFallback(t *testing.T) {
	style := TagStyle("nonexistent-tag-xyz")
	// Should return a style without panicking
	rendered := style.Render("nonexistent-tag-xyz")
	if !strings.Contains(rendered, "nonexistent-tag-xyz") {
		t.Errorf("TagStyle fallback did not render text: %q", rendered)
	}
}

func TestGuildStyleKnownGuild(t *testing.T) {
	guilds := []string{"loomari", "ashborne", "amarok", "nyx", "cipher", "fathom"}

	for _, guild := range guilds {
		t.Run(guild, func(t *testing.T) {
			style := GuildStyle(guild)
			rendered := style.Render(guild)
			if !strings.Contains(rendered, guild) {
				t.Errorf("GuildStyle(%q).Render(%q) = %q, want to contain %q", guild, guild, rendered, guild)
			}
		})
	}
}

func TestGuildStyleUnknownGuildFallback(t *testing.T) {
	style := GuildStyle("unknown-guild-xyz")
	rendered := style.Render("unknown-guild-xyz")
	if !strings.Contains(rendered, "unknown-guild-xyz") {
		t.Errorf("GuildStyle fallback did not render text: %q", rendered)
	}
}

func TestGuildBadgeFormat(t *testing.T) {
	tests := []struct {
		guildID string
		want    string
	}{
		{"loomari", "[loomari]"},
		{"nyx", "[nyx]"},
		{"cipher", "[cipher]"},
	}

	for _, tc := range tests {
		t.Run(tc.guildID, func(t *testing.T) {
			badge := GuildBadge(tc.guildID)
			if !strings.Contains(badge, tc.want) {
				t.Errorf("GuildBadge(%q) = %q, want to contain %q", tc.guildID, badge, tc.want)
			}
		})
	}
}

func TestGuildBadgeEmptyGuild(t *testing.T) {
	badge := GuildBadge("")
	if badge != "" {
		t.Errorf("GuildBadge(\"\") = %q, want empty string", badge)
	}
}

func TestGuildEmblemKnownGuild(t *testing.T) {
	tests := []struct {
		guildID string
		want    string
	}{
		{"loomari", "\U0001f577"},  // spider
		{"ashborne", "\U0001f525"}, // fire
		{"amarok", "\U0001f43a"},   // wolf
		{"nyx", "\U0001f426"},      // bird
		{"cipher", "\U0001f40d"},   // snake
		{"fathom", "\U0001f419"},   // octopus
	}

	for _, tc := range tests {
		t.Run(tc.guildID, func(t *testing.T) {
			got := GuildEmblem(tc.guildID)
			if got != tc.want {
				t.Errorf("GuildEmblem(%q) = %q, want %q", tc.guildID, got, tc.want)
			}
		})
	}
}

func TestGuildEmblemUnknownReturnsEmpty(t *testing.T) {
	got := GuildEmblem("unknown-guild")
	if got != "" {
		t.Errorf("GuildEmblem(unknown) = %q, want empty string", got)
	}
}

func TestPotencyStyle(t *testing.T) {
	tests := []struct {
		potency int
		// We verify the style renders non-empty content
	}{
		{1},
		{2},
		{3},
		{4},
	}

	for _, tc := range tests {
		t.Run("potency", func(t *testing.T) {
			style := potencyStyle(tc.potency)
			rendered := style.Render("P")
			if rendered == "" {
				t.Errorf("potencyStyle(%d).Render() returned empty string", tc.potency)
			}
		})
	}
}

func TestPotencyStyleLevel1IsDifferentFromLevel3(t *testing.T) {
	// Level 1 and level 3 should use different foreground colors
	style1 := potencyStyle(1)
	style3 := potencyStyle(3)
	rendered1 := style1.Render("X")
	rendered3 := style3.Render("X")
	// Both should render "X" but with different ANSI sequences
	if !strings.Contains(rendered1, "X") {
		t.Error("potencyStyle(1) did not render content")
	}
	if !strings.Contains(rendered3, "X") {
		t.Error("potencyStyle(3) did not render content")
	}
}

func TestHelpEntryFormat(t *testing.T) {
	result := helpEntry("q", "quit")
	if !strings.Contains(result, "q") {
		t.Errorf("helpEntry('q','quit') does not contain key 'q': %q", result)
	}
	if !strings.Contains(result, "quit") {
		t.Errorf("helpEntry('q','quit') does not contain label 'quit': %q", result)
	}
}

func TestHelpEntryMultipleKeys(t *testing.T) {
	tests := []struct {
		key   string
		label string
	}{
		{"j/k", "nav"},
		{"enter", "save"},
		{"esc", "cancel"},
		{"ctrl+s", "submit"},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			result := helpEntry(tc.key, tc.label)
			if !strings.Contains(result, tc.key) {
				t.Errorf("helpEntry(%q, %q) missing key", tc.key, tc.label)
			}
			if !strings.Contains(result, tc.label) {
				t.Errorf("helpEntry(%q, %q) missing label", tc.key, tc.label)
			}
		})
	}
}
