package tui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

// streamPollInterval is how often the home stream auto-refreshes.
const streamPollInterval = 15 * time.Second

type streamTickMsg time.Time

func streamTickCmd() tea.Cmd {
	return tea.Tick(streamPollInterval, func(t time.Time) tea.Msg {
		return streamTickMsg(t)
	})
}

type streamLoadedMsg struct {
	events []domain.StreamEvent
	err    error
}

type homeModel struct {
	client  *client.Client
	events  []domain.StreamEvent
	loading bool
	err     string
	width   int
	height  int
}

func newHomeModel(c *client.Client) homeModel {
	return homeModel{client: c}
}

func (m homeModel) Init() tea.Cmd {
	return m.load()
}

func (m homeModel) load() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		events, err := c.GetStream(context.Background(), false, pageSize, 0)
		return streamLoadedMsg{events: events, err: err}
	}
}

func (m homeModel) Update(msg tea.Msg) (homeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case streamLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.events = msg.events
			m.err = ""
		}
		return m, streamTickCmd()

	case streamTickMsg:
		return m, m.load()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m homeModel) View() string {
	if m.loading && len(m.events) == 0 {
		return " " + dimStyle.Render("loading stream...")
	}
	if m.err != "" {
		return " " + dimStyle.Render("error: "+m.err)
	}
	if len(m.events) == 0 {
		return " " + dimStyle.Render("no activity yet")
	}

	var sb strings.Builder

	// Legend line — colored dot + dim label, centered like the logo
	var legend strings.Builder
	legend.WriteString(forgeStyle.Render("●") + " " + dimStyle.Render("forge"))
	legend.WriteString("   ")
	legend.WriteString(castStyle.Render("●") + " " + dimStyle.Render("cast"))
	legend.WriteString("   ")
	legend.WriteString(joinStyle.Render("●") + " " + dimStyle.Render("join"))
	legend.WriteString("   ")
	legend.WriteString(rejectStyle.Render("●") + " " + dimStyle.Render("reject"))
	legend.WriteString("   ")
	legend.WriteString(goldStyle.Render("●") + " " + dimStyle.Render("grimoire"))
	legend.WriteString("   ")
	legend.WriteString(featuredStyle.Render("●") + " " + dimStyle.Render("featured"))
	legendStr := legend.String()
	legendWidth := lipgloss.Width(legendStr)
	legendPad := (m.width - legendWidth) / 2
	if legendPad < 0 {
		legendPad = 0
	}
	sb.WriteString(strings.Repeat(" ", legendPad) + legendStr + "\n")

	// Separator after legend (matches mockup border-bottom)
	sepW := m.width - 2
	if sepW < 4 {
		sepW = 4
	}
	sb.WriteString(" " + metaStyle.Render(strings.Repeat("─", sepW)) + "\n")

	// Count lines, not entries — each entry is body + optional voice + blank separator
	maxLines := m.height - 4 // legend(1) + separator(1) + input(1) + help(1)
	if maxLines < 5 {
		maxLines = 10
	}
	linesUsed := 0

	for i := 0; i < len(m.events) && linesUsed < maxLines; i++ {
		e := m.events[i]

		timeStr := formatTime(e.CreatedAt)

		var bar string
		var body string
		var voiceLine string // rendered as a second line when non-empty

		switch e.Kind {
		case "spell":
			bar = forgeStyle.Render("│")
			who := forgeStyle.Render(e.MagicianLogin)
			title := goldStyle.Render(`"` + truncStr(cleanTitle(e.Title), 60) + `"`)
			tagChip := ""
			if e.Tag != "" {
				tagChip = " — " + TagStyle(e.Tag).Render(e.Tag)
			}
			potChip := ""
			if e.Potency > 0 {
				potChip = " — " + potencyStyle(e.Potency).Render(fmt.Sprintf("P%d", e.Potency))
			}
			body = who + " forged " + title + tagChip + potChip
			if e.Voice != "" {
				voiceLine = grimLabelStyle.Render("Grimoire:") + " " + grimVoiceStyle.Render(e.Voice)
			}

		case "weapon":
			bar = castStyle.Render("│")
			title := castStyle.Bold(true).Render(`"` + truncStr(e.Title, 50) + `"`)
			body = title + " was cast."
			if e.Upvotes > 0 {
				body += " " + castStyle.Bold(true).Render(fmt.Sprintf("%d casts.", e.Upvotes))
			}
			if e.Voice != "" {
				voiceLine = grimLabelStyle.Render("Grimoire:") + " " + grimVoiceStyle.Render(e.Voice)
			}

		case "member":
			bar = joinStyle.Render("│")
			who := joinStyle.Render(e.MagicianLogin)
			guild := ""
			if e.GuildID != "" {
				guild = " " + GuildStyle(e.GuildID).Render(e.GuildID) + "."
			}
			body = who + " joined." + guild
			if e.Contributions > 0 || e.TopLanguage != "" {
				parts := []string{}
				if e.Contributions > 0 {
					parts = append(parts, fmt.Sprintf("%d contributions", e.Contributions))
				}
				if e.TopLanguage != "" {
					parts = append(parts, e.TopLanguage)
				}
				body += " " + dimStyle.Render(strings.Join(parts, " · "))
			}
			if e.Voice != "" {
				voiceLine = grimLabelStyle.Render("Grimoire:") + " " + grimVoiceStyle.Render(e.Voice)
			}

		case "reject":
			bar = rejectStyle.Render("│")
			title := rejectStyle.Render(`"` + truncStr(e.Title, 50) + `"`)
			body = "Submission rejected: " + title
			if e.Voice != "" {
				voiceLine = grimLabelStyle.Render("Grimoire:") + " " + voiceDimStyle.Render(e.Voice)
			}

		case "muse":
			bar = goldStyle.Render("│")
			body = grimLabelStyle.Render("Grimoire: ") + grimVoiceStyle.Render(e.Voice)

		case "featured":
			bar = featuredStyle.Render("│")
			name := featNameStyle.Render(e.MagicianLogin)
			project := featProjectStyle.Render(truncStr(e.Title, 40))
			insight := ""
			if e.Voice != "" && e.Tag != "" {
				insight = " " + featInsightStyle.Render(e.Voice) +
					" " + dimStyle.Render("("+e.Tag+")")
			} else if e.Voice != "" {
				insight = " " + featInsightStyle.Render(e.Voice)
			} else if e.Tag != "" {
				insight = " " + dimStyle.Render("("+e.Tag+")")
			}
			body = name + " is building " + project + " —" + insight

		case "convo":
			bar = normalStyle.Render("│")
			question := userMsgStyle.Render(truncStr(e.Title, 60))
			body = question
			if e.Voice != "" {
				voiceLine = grimLabelStyle.Render("Grimoire:") + " " + grimVoiceStyle.Render(e.Voice)
			}

		default:
			bar = metaStyle.Render("│")
			body = normalStyle.Render(e.Title)
		}

		// Prefix: " " + right-aligned time + "  " + colored bar + "  "
		prefix := " " + metaStyle.Render(fmt.Sprintf("%8s", timeStr)) + "  " + bar + "  "
		// Indent for continuation lines (aligns under body start)
		indent := strings.Repeat(" ", 14) // 1+8+2+1+2 = 14 visual chars

		// Available width for body/voice text in the body column
		bodyWidth := m.width - 15
		if bodyWidth < 20 {
			bodyWidth = 20
		}

		// Wrap body text so it stays within the body column
		wrappedBody := lipgloss.NewStyle().Width(bodyWidth).Render(body)
		bodyLines := strings.Split(wrappedBody, "\n")
		// First line gets the full prefix (time + bar)
		sb.WriteString(prefix + bodyLines[0] + "\n")
		linesUsed++
		// Continuation lines get space indent to align under body
		for _, bl := range bodyLines[1:] {
			sb.WriteString(" " + indent + bl + "\n")
			linesUsed++
		}

		if voiceLine != "" {
			// Wrap voice text to same width, cap at 3 lines
			wrapped := lipgloss.NewStyle().Width(bodyWidth).Render(voiceLine)
			vlines := strings.Split(wrapped, "\n")
			if len(vlines) > 3 {
				vlines = vlines[:3]
			}
			for _, vl := range vlines {
				sb.WriteString(" " + indent + vl + "\n")
				linesUsed++
			}
		}

		// Blank line between entries for visual breathing room
		sb.WriteString("\n")
		linesUsed++
	}

	return sb.String()
}

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
