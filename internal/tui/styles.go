package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Shimmer animation for the GRIMORA logo.
type shimmerTickMsg time.Time

func shimmerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return shimmerTickMsg(t)
	})
}

// renderShimmerLogo renders "G R I M O R A" as an endless flowing wave of green light.
// Deep forest green (#1a3a24) -> bright emerald (#4ade80). No hue drift.
// Letters are spaced apart (letter-spacing) and rendered without a background box.
func renderShimmerLogo(frame int) string {
	const text = "GRIMORA"
	n := len(text)

	var out string

	t := float64(frame)

	for i := 0; i < n; i++ {
		x := float64(i) / float64(n-1)

		// Flowing phase — one smooth wave advancing through the text
		phase := t*0.1 - x*3.0

		// Gentle speed modulation
		phase += math.Sin(t*0.023) * 2.0

		// Primary brightness wave
		b := math.Sin(phase)*0.5 + 0.5

		// Soft shaping
		b = math.Pow(b, 1.3)

		// Slow breathing tide
		tide := math.Sin(t*0.035) * 0.12
		b = b*0.75 + tide + 0.18

		if b > 1.0 {
			b = 1.0
		} else if b < 0.05 {
			b = 0.05
		}

		// Continuous RGB interpolation: deep forest green -> bright emerald
		// Deep:   (26, 58, 36)   #1a3a24
		// Bright: (74, 222, 128) #4ade80
		r := clampByte(26 + b*(74-26))
		g := clampByte(58 + b*(222-58))
		bl := clampByte(36 + b*(128-36))

		color := fmt.Sprintf("#%02X%02X%02X", r, g, bl)

		s := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(color))
		out += s.Render(string(text[i]))

		// Letter spacing — two spaces between each letter
		if i < n-1 {
			out += "  "
		}
	}

	return out
}

func clampByte(v float64) int {
	if v > 255 {
		return 255
	}
	if v < 0 {
		return 0
	}
	return int(v)
}

var (
	// Base styles — grimora neutral palette
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8890a0"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e4e4ec")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0c4d0"))

	metaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#505868"))

	// Help bar
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8890a0"))

	helpLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#505868"))

	// Search / accent
	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ade80")).
			Bold(true)

	upvoteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ade80"))

	// Accent / action styles
	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34d474"))

	// Grimoire voice styles
	grimVoiceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c8a84c")).
			Italic(true)

	grimLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d4a844")).
			Bold(true)

	// Event type styles
	forgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f59e0b"))

	castStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22d3ee"))

	rejectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#b45555"))

	goldStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d4a844"))

	// Surface colors
	borderColor  = lipgloss.Color("#1e1e2a")
	surfaceColor = lipgloss.Color("#111118")

	// Selected row background (matches mockup .grim-row.selected)
	selectedRowBg = lipgloss.NewStyle().Background(lipgloss.Color("#1e1e2a"))

	// Tag colors — from mockup CSS
	tagColors = map[string]lipgloss.Color{
		"debugging":     lipgloss.Color("#e06060"),
		"architecture":  lipgloss.Color("#b080d0"),
		"performance":   lipgloss.Color("#f0944a"),
		"code-review":   lipgloss.Color("#d4a844"),
		"testing":       lipgloss.Color("#60a0e0"),
		"security":      lipgloss.Color("#d05050"),
		"database":      lipgloss.Color("#3ecce4"),
		"ai-prompts":    lipgloss.Color("#c084e0"),
		"observability": lipgloss.Color("#8890a0"),
		// Carry forward tags not in mockup with distinct colors
		"refactoring":   lipgloss.Color("#b080d0"),
		"devops":        lipgloss.Color("#f0944a"),
		"data":          lipgloss.Color("#3ecce4"),
		"frontend":      lipgloss.Color("#d4a844"),
		"backend":       lipgloss.Color("#60a0e0"),
		"image-gen":     lipgloss.Color("#c084e0"),
		"writing":       lipgloss.Color("#f0944a"),
		"business":      lipgloss.Color("#b080d0"),
		"productivity":  lipgloss.Color("#8890a0"),
		"analysis":      lipgloss.Color("#3ecce4"),
		"system-prompt": lipgloss.Color("#d4a844"),
		"education":     lipgloss.Color("#c084e0"),
		"coding":        lipgloss.Color("#e4e4ec"),
		"conversation":  lipgloss.Color("#8890a0"),
		"general":       lipgloss.Color("#606878"),
	}

	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#606878"))

	commentTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#606878"))

	commentTimeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#505868"))

	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#34d474")).
				Bold(true)

	inputPlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#343c4a"))

	// Mention styles (Hall)
	mentionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ade80")).
			Bold(true)

	mentionSelfStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#86efac")).
				Bold(true)

	// Chat styles (Hall)
	chatSelfNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#e4e4ec"))

	chatInputNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4ade80")) // emerald — distinguishes input from chat

	chatSelfTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c0c4d0"))

	chatTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8890a0"))

	chatSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#404858"))

	chatSysStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#404858"))

	presenceTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8890a0")).
				Bold(true)

	presenceDotStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#34d474"))

	// Guild colors — from mockup .g-* classes
	guildColors = map[string]lipgloss.Color{
		"loomari":  lipgloss.Color("#43e88c"),
		"ashborne": lipgloss.Color("#f0944a"),
		"amarok":   lipgloss.Color("#b8ccdf"),
		"nyx":      lipgloss.Color("#c084e0"),
		"cipher":   lipgloss.Color("#34d474"),
		"fathom":   lipgloss.Color("#3ecce4"),
	}
)

// guildEmblems maps guild IDs to emoji emblems.
var guildEmblems = map[string]string{
	"loomari":  "\U0001f577", // spider
	"ashborne": "\U0001f525", // fire
	"amarok":   "\U0001f43a", // wolf
	"nyx":      "\U0001f426", // bird
	"cipher":   "\U0001f40d", // snake
	"fathom":   "\U0001f419", // octopus
}

// GuildEmblem returns the emoji emblem for a guild ID.
func GuildEmblem(guildID string) string {
	if e, ok := guildEmblems[guildID]; ok {
		return e
	}
	return ""
}

// TagStyle returns a bold style colored for the given tag.
func TagStyle(tag string) lipgloss.Style {
	if c, ok := tagColors[tag]; ok {
		return lipgloss.NewStyle().Foreground(c).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#606878")).Bold(true)
}

// GuildStyle returns a bold style colored for the given guild ID.
func GuildStyle(guildID string) lipgloss.Style {
	if c, ok := guildColors[guildID]; ok {
		return lipgloss.NewStyle().Foreground(c).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#8890a0")).Bold(true)
}

// GuildBadge returns a short colored badge string for a guild, e.g. "[NYX]".
func GuildBadge(guildID string) string {
	if guildID == "" {
		return ""
	}
	label := "[" + guildID + "]"
	return GuildStyle(guildID).Render(label)
}

// rankStyle returns a colored style based on leaderboard position.
func rankStyle(rank int) lipgloss.Style {
	switch rank {
	case 1:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa")) // blue
	case 2:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171")) // red
	case 3:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#facc15")) // yellow
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8891a5")) // soft slate
	}
}

// potencyStyle returns a style for the given potency level.
func potencyStyle(potency int) lipgloss.Style {
	switch {
	case potency >= 3:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Bold(true)
	case potency == 2:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24")).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#92711a"))
	}
}

// cardBorder renders animated top or bottom border for rich message cards.
// pos: "top" or "bottom". label: optional header text (top only).
// baseColor: hex color. frame: 0=static, 1-20=animating. width: terminal width.
func cardBorder(pos, label, baseColor string, frame, width int) string {
	w := width - 4
	if w < 10 {
		w = 10
	}

	if pos == "bottom" {
		border := " └" + strings.Repeat("─", w)
		if frame > 0 && frame <= 20 {
			return animBorderLine(border, baseColor, frame, w)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(baseColor)).Render(border)
	}

	// Top border with label
	var header string
	if label != "" {
		header = " ┌ " + label + " "
		remaining := w - lipgloss.Width(header) + 2 // +2 for " ┌"
		if remaining < 1 {
			remaining = 1
		}
		header += lipgloss.NewStyle().Foreground(lipgloss.Color(baseColor)).Render(strings.Repeat("─", remaining))
	} else {
		header = " ┌" + strings.Repeat("─", w)
	}

	if frame > 0 && frame <= 20 && label == "" {
		return animBorderLine(header, baseColor, frame, w)
	}
	// Label cards: animate just the border dashes
	if frame > 0 && frame <= 20 {
		remaining := w - lipgloss.Width(" ┌ "+label+" ") + 2
		if remaining < 1 {
			remaining = 1
		}
		prefix := " ┌ " + label + " "
		return prefix + animBorderDashes(remaining, baseColor, frame)
	}
	return header
}

// animBorderLine renders a full border line with sine-wave brightness animation.
func animBorderLine(line, baseColor string, frame, width int) string {
	r0, g0, b0 := hexToRGB(baseColor)
	// Dim version: 40% brightness
	rD, gD, bD := int(float64(r0)*0.4), int(float64(g0)*0.4), int(float64(b0)*0.4)

	t := float64(frame)
	var out string
	for i, ch := range line {
		x := float64(i) / float64(max(width, 1))
		phase := t*0.3 - x*4.0
		b := math.Sin(phase)*0.5 + 0.5
		b = math.Pow(b, 1.5)
		r := clampByte(float64(rD) + b*float64(r0-rD))
		g := clampByte(float64(gD) + b*float64(g0-gD))
		bl := clampByte(float64(bD) + b*float64(b0-bD))
		c := fmt.Sprintf("#%02X%02X%02X", r, g, bl)
		out += lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(string(ch))
	}
	return out
}

// animBorderDashes renders N dashes with sine-wave brightness.
func animBorderDashes(n int, baseColor string, frame int) string {
	r0, g0, b0 := hexToRGB(baseColor)
	rD, gD, bD := int(float64(r0)*0.4), int(float64(g0)*0.4), int(float64(b0)*0.4)

	t := float64(frame)
	var out string
	for i := 0; i < n; i++ {
		x := float64(i) / float64(max(n, 1))
		phase := t*0.3 - x*4.0
		b := math.Sin(phase)*0.5 + 0.5
		b = math.Pow(b, 1.5)
		r := clampByte(float64(rD) + b*float64(r0-rD))
		g := clampByte(float64(gD) + b*float64(g0-gD))
		bl := clampByte(float64(bD) + b*float64(b0-bD))
		c := fmt.Sprintf("#%02X%02X%02X", r, g, bl)
		out += lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("─")
	}
	return out
}

// hexToRGB parses a hex color string (#RRGGBB) into r,g,b ints.
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 128, 128, 128
	}
	var r, g, b int
	_, _ = fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b) //nolint:errcheck
	return r, g, b
}

// helpEntry renders a single "key label" pair for help bars.
func helpEntry(key, label string) string {
	return helpKeyStyle.Render(key) + " " + helpLabelStyle.Render(label)
}

// helpItem is a selectable link in the help overlay.
type helpItem struct {
	label string
	desc  string
	url   string
}

var helpItems = []helpItem{
	{"Terms of Service", "grimora.ai/terms", "https://grimora.ai/terms"},
	{"Privacy Policy", "grimora.ai/privacy", "https://grimora.ai/privacy"},
	{"FAQ", "grimora.ai/faq", "https://grimora.ai/faq"},
	{"Website", "grimora.ai", "https://grimora.ai"},
}

// helpView renders the interactive help overlay with a cursor.
func helpView(cursor int) string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ade80")).
		Bold(true).
		Render("G R I M O R A")

	quote := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Render(`"The Grimoire sees all. Here is what it permits."`)

	attrib := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D4A017")).
		Render("— The Grimoire")

	cmdStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4ade80"))
	linkDescStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)

	commands := []struct{ cmd, desc string }{
		{"grimora", "Enter the Hall (interactive TUI)"},
		{"grimora login", "Authenticate with GitHub"},
		{"grimora logout", "Clear your session"},
		{"grimora update", "Check for updates"},
		{"grimora --version", "Show version"},
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s\n\n  %s\n  %s\n\n", title, quote, attrib)

	// Commands section
	fmt.Fprintf(&b, "  %s\n", sectionStyle.Render("Commands"))
	for _, c := range commands {
		fmt.Fprintf(&b, "    %s  %s\n", cmdStyle.Render(fmt.Sprintf("%-20s", c.cmd)), descStyle.Render(c.desc))
	}

	// Links section (selectable)
	fmt.Fprintf(&b, "\n  %s\n", sectionStyle.Render("Links (enter to open)"))
	for i, item := range helpItems {
		label := cmdStyle.Render(fmt.Sprintf("%-20s", item.label))
		prefix := "    "
		if i == cursor {
			label = selectedStyle.Render(fmt.Sprintf("%-20s", item.label))
			prefix = "  > "
		}
		fmt.Fprintf(&b, "%s%s  %s\n", prefix, label, linkDescStyle.Render(item.desc))
	}
	return b.String()
}
