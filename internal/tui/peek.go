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

type peekModel struct {
	client   *client.Client
	card     *domain.MagicianCard
	projects []domain.WorkshopProject
	closed   bool
	err      string
	width    int
}

func newPeekModel(c *client.Client) peekModel {
	return peekModel{client: c}
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

	// Workshop section
	if len(m.projects) > 0 {
		sb.WriteString("\n" + sectionHeaderStyle.Render("── WORKSHOP ──") + "\n")
		for _, p := range m.projects {
			sb.WriteString("  " + normalStyle.Render(p.Name) + "\n")
			if p.Insight != "" {
				sb.WriteString("    " + grimVoiceStyle.Render(p.Insight) + "\n")
			}
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
