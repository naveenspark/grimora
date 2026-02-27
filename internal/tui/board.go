package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

// -- messages --

type boardLoadedMsg struct {
	entries []domain.LeaderboardEntry
	err     error
}

type boardFollowMsg struct {
	login string
	err   error
}

// -- model --

type boardModel struct {
	client      *client.Client
	entries     []domain.LeaderboardEntry
	cursor      int
	guildFilter string // "" = all, else guild id
	guildCycle  int    // index into guildOrder for cycling
	cityFilter  string // "" = all, else city name
	cityCycle   int    // index into cityOrder for cycling
	cityOrder   []string
	err         string
	loading     bool
	myLogin     string
	width       int
	height      int
}

// guildOrder is the cycle order for guild filtering.
var guildOrder = []string{"", "loomari", "ashborne", "amarok", "nyx", "cipher", "fathom"}

func newBoardModel(c *client.Client) boardModel {
	return boardModel{client: c}
}

func (m *boardModel) buildCityOrder() {
	seen := make(map[string]bool)
	for _, e := range m.entries {
		if e.City != "" {
			seen[e.City] = true
		}
	}
	cities := make([]string, 0, len(seen))
	for c := range seen {
		cities = append(cities, c)
	}
	sort.Strings(cities)
	m.cityOrder = append([]string{""}, cities...) // "" = all
	m.cityCycle = 0
	m.cityFilter = ""
}

func (m boardModel) Init() tea.Cmd {
	return m.loadBoard()
}

func (m boardModel) loadBoard() tea.Cmd {
	c := m.client
	guild := m.guildFilter
	city := m.cityFilter
	return func() tea.Msg {
		entries, err := c.GetLeaderboard(context.Background(), guild, city, 50, 0)
		return boardLoadedMsg{entries: entries, err: err}
	}
}

func (m boardModel) Update(msg tea.Msg) (boardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case meLoadedMsg:
		if msg.err == nil && msg.me != nil {
			m.myLogin = msg.me.GitHubLogin
		}

	case boardLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.entries = msg.entries
			m.err = ""
			if m.cursor >= len(m.entries) {
				m.cursor = 0
			}
			if m.cityFilter == "" {
				m.buildCityOrder()
			}
		}

	case boardFollowMsg:
		// Refresh after follow action
		if msg.err == nil {
			return m, m.loadBoard()
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m boardModel) handleKey(msg tea.KeyMsg) (boardModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g":
		// Cycle guild filter
		m.guildCycle = (m.guildCycle + 1) % len(guildOrder)
		m.guildFilter = guildOrder[m.guildCycle]
		m.cursor = 0
		m.loading = true
		return m, m.loadBoard()
	case "c":
		if len(m.cityOrder) > 1 {
			m.cityCycle = (m.cityCycle + 1) % len(m.cityOrder)
			m.cityFilter = m.cityOrder[m.cityCycle]
			m.cursor = 0
			m.loading = true
			return m, m.loadBoard()
		}
	case "p":
		if len(m.entries) > 0 && m.cursor < len(m.entries) {
			login := m.entries[m.cursor].Login
			return m, func() tea.Msg { return showPeekMsg{login: login} }
		}
	case "f":
		if len(m.entries) > 0 && m.cursor < len(m.entries) {
			login := m.entries[m.cursor].Login
			c := m.client
			return m, func() tea.Msg {
				err := c.Follow(context.Background(), login)
				return boardFollowMsg{login: login, err: err}
			}
		}
	case "r":
		m.loading = true
		return m, m.loadBoard()
	}
	return m, nil
}

func (m boardModel) View() string {
	var b strings.Builder

	// Filter line (only show if a filter is active)
	if m.guildFilter != "" || m.cityFilter != "" {
		parts := []string{}
		if m.guildFilter != "" {
			parts = append(parts, GuildStyle(m.guildFilter).Render(m.guildFilter))
		}
		if m.cityFilter != "" {
			parts = append(parts, dimStyle.Render(m.cityFilter))
		}
		b.WriteString(" " + strings.Join(parts, dimStyle.Render(" · ")) + "\n")
	}

	if m.loading && len(m.entries) == 0 {
		b.WriteString(" " + dimStyle.Render("loading...") + "\n")
		return b.String()
	}
	if m.err != "" {
		b.WriteString(" " + dimStyle.Render("error: "+m.err) + "\n")
		return b.String()
	}
	if len(m.entries) == 0 {
		b.WriteString("\n " + dimStyle.Render("the board is empty — be the first to forge a spell") + "\n")
		return b.String()
	}

	for i, entry := range m.entries {
		isActive := i == m.cursor
		isYou := m.myLogin != "" && entry.Login == m.myLogin

		cursor := " "
		if isActive {
			cursor = accentStyle.Render("▸")
		}

		rankLabel := fmt.Sprintf("#%-3d", entry.Rank)
		rankStr := rankStyle(entry.Rank).Render(rankLabel)
		if isYou {
			rankStr = accentStyle.Render(rankLabel)
		}

		var loginStyled string
		if isYou {
			loginStyled = selectedStyle.Render(fmt.Sprintf("%-16s", "you"))
		} else {
			loginStyled = GuildStyle(entry.GuildID).Render(fmt.Sprintf("%-16s", entry.Login))
		}

		spells := metaStyle.Render(fmt.Sprintf("%d spells", entry.SpellsForged))

		potencyStr := ""
		if entry.TotalPotency > 0 {
			potencyStr = potencyStyle(entry.TotalPotency).Render(fmt.Sprintf("P%d", entry.TotalPotency))
		}

		cityStr := ""
		if entry.City != "" {
			cityStr = dimStyle.Render(entry.City)
		}

		youMarker := ""
		if isYou {
			youMarker = " " + accentStyle.Render("<- you")
		}

		row := fmt.Sprintf(" %s %s  %s  %s", cursor, rankStr, loginStyled, spells)
		if potencyStr != "" {
			row += "  " + potencyStr
		}
		if cityStr != "" {
			row += "  " + cityStr
		}
		row += youMarker + "\n"
		b.WriteString(row)
	}

	// Filter hint
	filterHint := dimStyle.Render("g cycle guild · c cycle city")
	b.WriteString("\n " + filterHint + "\n")

	return b.String()
}

func (m boardModel) helpKeys() string {
	return helpEntry("j/k", "nav") + "  " + helpEntry("g", "guild") + "  " + helpEntry("c", "city") + "  " + helpEntry("p", "peek") + "  " + helpEntry("f", "follow") + "  " + helpEntry("r", "refresh") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
}
