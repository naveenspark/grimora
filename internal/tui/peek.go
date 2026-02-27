package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

type peekLoadedMsg struct {
	card *domain.MagicianCard
	err  error
}

type peekWorkshopMsg struct {
	projects []domain.WorkshopProject
	err      error
}

type peekFollowMsg struct {
	login string
	err   error
}

type peekProjectUpdatesMsg struct {
	projectID string
	updates   []domain.ProjectUpdate
	err       error
}

type peekModel struct {
	client         *client.Client
	card           *domain.MagicianCard
	projects       []domain.WorkshopProject
	projectUpdates map[string][]domain.ProjectUpdate
	closed         bool
	err            string
	width          int
}

func newPeekModel(c *client.Client) peekModel {
	return peekModel{client: c, projectUpdates: make(map[string][]domain.ProjectUpdate)}
}

func (m peekModel) load(login string) tea.Cmd {
	c := m.client
	cardCmd := func() tea.Msg {
		card, err := c.GetMagician(context.Background(), login)
		if err != nil {
			return peekLoadedMsg{err: fmt.Errorf("client.GetMagician: %w", err)}
		}
		return peekLoadedMsg{card: card}
	}
	workshopCmd := func() tea.Msg {
		projects, err := c.GetMagicianWorkshop(context.Background(), login)
		if err != nil {
			return peekWorkshopMsg{err: fmt.Errorf("client.GetMagicianWorkshop: %w", err)}
		}
		return peekWorkshopMsg{projects: projects}
	}
	return tea.Batch(cardCmd, workshopCmd)
}

func (m peekModel) Update(msg tea.Msg) (peekModel, tea.Cmd) {
	switch msg := msg.(type) {
	case peekLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.card = msg.card
		}
		return m, nil

	case peekWorkshopMsg:
		if msg.err == nil {
			m.projects = msg.projects
		}
		// Load project updates for timelines
		var cmds []tea.Cmd
		if m.client != nil {
			for _, p := range m.projects {
				pid := p.ID.String()
				c := m.client
				cmds = append(cmds, func() tea.Msg {
					updates, err := c.ListProjectUpdates(context.Background(), pid)
					return peekProjectUpdatesMsg{projectID: pid, updates: updates, err: err}
				})
			}
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case peekProjectUpdatesMsg:
		if msg.err == nil {
			m.projectUpdates[msg.projectID] = msg.updates
		}
		return m, nil

	case peekFollowMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else if m.card != nil {
			m.card.IsFollowing = !m.card.IsFollowing
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.closed = true
		case "f":
			if m.card != nil {
				login := m.card.GitHubLogin
				isFollowing := m.card.IsFollowing
				c := m.client
				return m, func() tea.Msg {
					var err error
					if isFollowing {
						err = c.Unfollow(context.Background(), login)
					} else {
						err = c.Follow(context.Background(), login)
					}
					return peekFollowMsg{login: login, err: err}
				}
			}
		}
	}
	return m, nil
}

func (m peekModel) View() string {
	if m.err != "" {
		return "\n " + dimStyle.Render("peek error: "+m.err)
	}
	if m.card == nil {
		return "\n " + dimStyle.Render("loading...")
	}

	card := m.card
	cardWidth := min(50, m.width-4)
	if cardWidth < 30 {
		cardWidth = 30
	}
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Background(surfaceColor).
		Padding(1, 2).
		Width(cardWidth)

	var sb strings.Builder

	// Emblem + name
	emblem := GuildEmblem(card.GuildID)
	if emblem != "" {
		sb.WriteString(emblem + " ")
	}
	sb.WriteString(selectedStyle.Render(card.GitHubLogin) + "\n")

	// Guild + presence dot + city
	if card.GuildID != "" {
		sb.WriteString("   " + GuildStyle(card.GuildID).Render(card.GuildID))
	}
	if card.Online {
		sb.WriteString("  " + presenceDotStyle.Render("●") + " " + presenceDotStyle.Render("online"))
	} else {
		sb.WriteString("  " + dimStyle.Render("○") + " " + dimStyle.Render("offline"))
	}
	if card.City != "" {
		sb.WriteString(" · " + metaStyle.Render(card.City))
	}
	sb.WriteString("\n")

	// Stats
	sb.WriteString(metaStyle.Render("---") + "\n")
	stats := fmt.Sprintf("#%d rank  %d spells  P%d potency",
		card.CardNumber, card.SpellCount, card.TotalPotency)
	sb.WriteString(metaStyle.Render(stats) + "\n")
	sb.WriteString(metaStyle.Render("---") + "\n")

	// Workshop section with timelines
	if len(m.projects) > 0 {
		sb.WriteString("\n" + sectionHeaderStyle.Render("── BUILD JOURNAL ──") + "\n")
		for _, p := range m.projects {
			updates := m.projectUpdates[p.ID.String()]
			status := projectStatus(updates)
			badge := dimStyle.Render("building")
			if status == "shipped" {
				badge = goldStyle.Render("shipped")
			}
			nameW := lipgloss.Width("  " + p.Name)
			badgeW := lipgloss.Width(badge)
			innerW := cardWidth - 4 // account for border padding
			padLen := innerW - nameW - badgeW
			if padLen < 2 {
				padLen = 2
			}
			sb.WriteString("  " + normalStyle.Render(p.Name) + strings.Repeat(" ", padLen) + badge + "\n")
			if p.Insight != "" {
				sb.WriteString("    " + dimStyle.Render(p.Insight) + "\n")
			}
			// Timeline
			if len(updates) > 0 {
				sb.WriteString("    " + dimStyle.Render("│") + "\n")
				maxShow := 3
				start := 0
				if len(updates) > maxShow {
					start = len(updates) - maxShow
				}
				for j := start; j < len(updates); j++ {
					u := updates[j]
					ts := metaStyle.Render(formatTime(u.CreatedAt))
					switch u.Kind {
					case "start":
						sb.WriteString("    " + accentStyle.Render("●") + " " + accentStyle.Render("started") + "  " + ts + "\n")
					case "ship":
						sb.WriteString("    " + goldStyle.Render("✦") + " " + goldStyle.Render("shipped") + "  " + ts + "\n")
						if u.Body != "" {
							sb.WriteString("    " + dimStyle.Render("│") + " " + dimStyle.Render(u.Body) + "\n")
						}
					default:
						body := u.Body
						if len([]rune(body)) > 30 {
							body = string([]rune(body)[:29]) + "…"
						}
						sb.WriteString("    " + dimStyle.Render("●") + " " + dimStyle.Render(body) + "  " + ts + "\n")
					}
					if j < len(updates)-1 {
						sb.WriteString("    " + dimStyle.Render("│") + "\n")
					}
				}
				if status != "shipped" {
					sb.WriteString("\n    " + dimStyle.Render("○ ···") + "\n")
				}
			}
			sb.WriteString("\n")
		}
	}

	// Follow action hint
	sb.WriteString("\n")
	if card.IsFollowing {
		sb.WriteString(accentStyle.Render("following") + "  " + helpKeyStyle.Render("f") + " " + helpLabelStyle.Render("unfollow"))
	} else {
		sb.WriteString(helpKeyStyle.Render("f") + " " + helpLabelStyle.Render("follow"))
	}
	sb.WriteString("  " + helpKeyStyle.Render("esc") + " " + helpLabelStyle.Render("close"))

	return "\n" + border.Render(sb.String())
}
