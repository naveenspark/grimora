package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

// browsing is true when the full browse panel is shown instead of the unified view.

// workshopState is the state machine for workshop CRUD interactions.
type workshopState int

const (
	wsNormal   workshopState = iota
	wsEditing                // editing insight of selected project
	wsAdding                 // adding new project (name + insight fields)
	wsDeleting               // delete confirmation
)

// -- messages --

type youLoadedMsg struct {
	cards []domain.MagicianCard
	err   error
}

type youInvitesLoadedMsg struct {
	invites []domain.Invite
	err     error
}

type youFollowMsg struct {
	login string
	err   error
}

type youCopyMsg struct{ err error }

type workshopLoadedMsg struct {
	projects []domain.WorkshopProject
	err      error
}

type workshopCreatedMsg struct {
	project *domain.WorkshopProject
	err     error
}

type workshopUpdatedMsg struct{ err error }

type workshopDeletedMsg struct {
	id  string
	err error
}

// -- model --

// youSection identifies which section the cursor is in.
type youSection int

const (
	youSectionWorkshop youSection = iota
	youSectionInvites
	youSectionLeaderboard
)

// inviteSpellThreshold is the number of forged spells required for the next invite code.
const inviteSpellThreshold = 10

type youModel struct {
	client    *client.Client
	cards     []domain.MagicianCard
	invites   []domain.Invite
	me        *domain.Magician
	cursor    int // leaderboard cursor
	browsing  bool
	loading   bool
	err       string
	statusMsg string
	width     int
	height    int
	section   youSection // active section for navigation

	// workshop
	projects     []domain.WorkshopProject
	wsCursor     int
	wsState      workshopState
	wsEditText   string // insight being edited
	wsAddName    string // name field when adding
	wsAddInsight string // insight field when adding
	wsAddFocus   int    // 0=name, 1=insight when adding

	// invites
	inviteCursor int
}

func newYouModel(c *client.Client) youModel {
	return youModel{client: c}
}

func (m youModel) Init() tea.Cmd {
	return tea.Batch(m.loadMagicians(), m.loadInvites(), m.loadWorkshop())
}

func (m youModel) loadMagicians() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		cards, err := c.ListMagicians(context.Background(), pageSize, 0)
		return youLoadedMsg{cards: cards, err: err}
	}
}

func (m youModel) loadInvites() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		invites, err := c.ListInvites(context.Background())
		return youInvitesLoadedMsg{invites: invites, err: err}
	}
}

func (m youModel) loadWorkshop() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		projects, err := c.ListWorkshopProjects(context.Background())
		return workshopLoadedMsg{projects: projects, err: err}
	}
}

func (m youModel) Update(msg tea.Msg) (youModel, tea.Cmd) {
	switch msg := msg.(type) {
	case youLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.cards = msg.cards
			m.err = ""
			if m.cursor >= len(m.cards) {
				m.cursor = 0
			}
		}
		return m, nil

	case youInvitesLoadedMsg:
		if msg.err == nil {
			m.invites = msg.invites
		}
		return m, nil

	case meLoadedMsg:
		if msg.err == nil && msg.me != nil {
			m.me = msg.me
		}
		return m, nil

	case workshopLoadedMsg:
		if msg.err == nil {
			m.projects = msg.projects
			if m.wsCursor >= len(m.projects) {
				m.wsCursor = 0
			}
		}
		return m, nil

	case workshopCreatedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("create failed: %v", msg.err)
		} else {
			if msg.project != nil {
				m.projects = append(m.projects, *msg.project)
				m.wsCursor = len(m.projects) - 1
			}
			m.statusMsg = "project added"
		}
		m.wsState = wsNormal
		m.wsAddName = ""
		m.wsAddInsight = ""
		m.wsAddFocus = 0
		return m, nil

	case workshopUpdatedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("update failed: %v", msg.err)
		} else {
			m.statusMsg = "saved"
		}
		m.wsState = wsNormal
		m.wsEditText = ""
		return m, nil

	case workshopDeletedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("delete failed: %v", msg.err)
		} else {
			// Remove from local slice
			for i, p := range m.projects {
				if p.ID.String() == msg.id {
					m.projects = append(m.projects[:i], m.projects[i+1:]...)
					break
				}
			}
			if m.wsCursor >= len(m.projects) && m.wsCursor > 0 {
				m.wsCursor = len(m.projects) - 1
			}
			m.statusMsg = "project removed"
		}
		m.wsState = wsNormal
		return m, nil

	case youFollowMsg:
		if msg.err != nil {
			if strings.Contains(msg.err.Error(), "not authenticated") {
				m.statusMsg = "not authenticated -- run: grimora login"
			} else {
				m.statusMsg = fmt.Sprintf("follow failed: %v", msg.err)
			}
		} else {
			for i := range m.cards {
				if m.cards[i].GitHubLogin == msg.login {
					m.cards[i].IsFollowing = !m.cards[i].IsFollowing
					break
				}
			}
			m.statusMsg = ""
		}
		return m, nil

	case youCopyMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("copy failed: %v", msg.err)
		} else {
			m.statusMsg = "copied!"
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		m.statusMsg = ""
		return m.handleKey(msg)
	}
	return m, nil
}

func (m youModel) handleKey(msg tea.KeyMsg) (youModel, tea.Cmd) {
	// Workshop state machine intercepts keys first when active
	switch m.wsState {
	case wsEditing:
		return m.handleKeyEditing(msg)
	case wsAdding:
		return m.handleKeyAdding(msg)
	case wsDeleting:
		return m.handleKeyDeleting(msg)
	}

	// Normal mode
	switch msg.String() {
	case "j", "down":
		if m.browsing {
			if m.wsCursor < len(m.projects)-1 {
				m.wsCursor++
			}
		} else {
			m.navDown()
		}

	case "k", "up":
		if m.browsing {
			if m.wsCursor > 0 {
				m.wsCursor--
			}
		} else {
			m.navUp()
		}

	case "b":
		m.browsing = !m.browsing

	case "e":
		// Edit insight of selected workshop project
		if len(m.projects) > 0 && m.wsCursor < len(m.projects) {
			m.wsState = wsEditing
			m.wsEditText = m.projects[m.wsCursor].Insight
		}

	case "a":
		// Add new workshop project
		m.wsState = wsAdding
		m.wsAddName = ""
		m.wsAddInsight = ""
		m.wsAddFocus = 0

	case "d":
		// Delete selected workshop project
		if len(m.projects) > 0 && m.wsCursor < len(m.projects) {
			m.wsState = wsDeleting
		}

	case "f":
		// Follow/unfollow — uses cursor for leaderboard card index
		if len(m.cards) > 0 && m.cursor < len(m.cards) {
			card := m.cards[m.cursor]
			login := card.GitHubLogin
			isFollowing := card.IsFollowing
			c := m.client
			return m, func() tea.Msg {
				var err error
				if isFollowing {
					err = c.Unfollow(context.Background(), login)
				} else {
					err = c.Follow(context.Background(), login)
				}
				return youFollowMsg{login: login, err: err}
			}
		}

	case "c":
		// Copy the selected (or first available) invite code
		avail := m.availableInvites()
		idx := 0
		if m.section == youSectionInvites && m.inviteCursor < len(avail) {
			idx = m.inviteCursor
		}
		if idx < len(avail) {
			code := avail[idx].Code
			return m, func() tea.Msg {
				err := clipboard.WriteAll("grimora.ai/join/" + code)
				return youCopyMsg{err: err}
			}
		}

	case "r":
		m.loading = true
		return m, tea.Batch(m.loadMagicians(), m.loadInvites(), m.loadWorkshop())

	case "p":
		if len(m.cards) > 0 && m.cursor < len(m.cards) {
			login := m.cards[m.cursor].GitHubLogin
			return m, func() tea.Msg { return showPeekMsg{login: login} }
		}
	}
	return m, nil
}

func (m youModel) handleKeyEditing(msg tea.KeyMsg) (youModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Save
		if len(m.projects) > 0 && m.wsCursor < len(m.projects) {
			proj := m.projects[m.wsCursor]
			id := proj.ID.String()
			name := proj.Name
			insight := m.wsEditText
			// Optimistic update
			m.projects[m.wsCursor].Insight = insight
			c := m.client
			return m, func() tea.Msg {
				err := c.UpdateWorkshopProject(context.Background(), id, name, insight)
				return workshopUpdatedMsg{err: err}
			}
		}
		m.wsState = wsNormal
	case "esc":
		m.wsState = wsNormal
		m.wsEditText = ""
	default:
		m.wsEditText = editRune(m.wsEditText, msg.String())
	}
	return m, nil
}

func (m youModel) handleKeyAdding(msg tea.KeyMsg) (youModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.wsAddFocus = 1 - m.wsAddFocus // toggle between 0 and 1
	case "enter":
		name := strings.TrimSpace(m.wsAddName)
		insight := strings.TrimSpace(m.wsAddInsight)
		if name == "" {
			m.statusMsg = "name required"
			return m, nil
		}
		c := m.client
		return m, func() tea.Msg {
			project, err := c.CreateWorkshopProject(context.Background(), name, insight)
			return workshopCreatedMsg{project: project, err: err}
		}
	case "esc":
		m.wsState = wsNormal
		m.wsAddName = ""
		m.wsAddInsight = ""
		m.wsAddFocus = 0
	default:
		if m.wsAddFocus == 0 {
			m.wsAddName = editRune(m.wsAddName, msg.String())
		} else {
			m.wsAddInsight = editRune(m.wsAddInsight, msg.String())
		}
	}
	return m, nil
}

func (m youModel) handleKeyDeleting(msg tea.KeyMsg) (youModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if len(m.projects) > 0 && m.wsCursor < len(m.projects) {
			id := m.projects[m.wsCursor].ID.String()
			c := m.client
			return m, func() tea.Msg {
				err := c.DeleteWorkshopProject(context.Background(), id)
				return workshopDeletedMsg{id: id, err: err}
			}
		}
		m.wsState = wsNormal
	case "n", "N", "esc":
		m.wsState = wsNormal
	}
	return m, nil
}

// availableInvites returns invites that haven't been claimed.
func (m youModel) availableInvites() []domain.Invite {
	var out []domain.Invite
	for _, inv := range m.invites {
		if inv.UsedBy == nil {
			out = append(out, inv)
		}
	}
	return out
}

// navDown moves the cursor down within or across sections.
func (m *youModel) navDown() {
	switch m.section {
	case youSectionWorkshop:
		if m.wsCursor < len(m.projects)-1 {
			m.wsCursor++
		} else if len(m.availableInvites()) > 0 {
			m.section = youSectionInvites
			m.inviteCursor = 0
		} else if len(m.cards) > 0 {
			m.section = youSectionLeaderboard
			m.cursor = 0
		}
	case youSectionInvites:
		avail := m.availableInvites()
		if m.inviteCursor < len(avail)-1 {
			m.inviteCursor++
		} else if len(m.cards) > 0 {
			m.section = youSectionLeaderboard
			m.cursor = 0
		}
	case youSectionLeaderboard:
		maxRows := len(m.cards)
		if maxRows > 10 {
			maxRows = 10
		}
		if m.cursor < maxRows-1 {
			m.cursor++
		}
	}
}

// navUp moves the cursor up within or across sections.
func (m *youModel) navUp() {
	switch m.section {
	case youSectionWorkshop:
		if m.wsCursor > 0 {
			m.wsCursor--
		}
	case youSectionInvites:
		if m.inviteCursor > 0 {
			m.inviteCursor--
		} else if len(m.projects) > 0 {
			m.section = youSectionWorkshop
			m.wsCursor = len(m.projects) - 1
		} else {
			m.section = youSectionWorkshop
		}
	case youSectionLeaderboard:
		if m.cursor > 0 {
			m.cursor--
		} else if len(m.availableInvites()) > 0 {
			m.section = youSectionInvites
			m.inviteCursor = len(m.availableInvites()) - 1
		} else {
			m.section = youSectionWorkshop
			if len(m.projects) > 0 {
				m.wsCursor = len(m.projects) - 1
			}
		}
	}
}

// helpKeys returns context-sensitive help text based on the current state.
func (m youModel) helpKeys() string {
	switch m.wsState {
	case wsEditing:
		return " " + helpEntry("enter", "save") + "  " + helpEntry("esc", "cancel")
	case wsAdding:
		return " " + helpEntry("tab", "next") + "  " + helpEntry("enter", "save") + "  " + helpEntry("esc", "cancel")
	case wsDeleting:
		return " " + helpEntry("y", "confirm") + "  " + helpEntry("n", "cancel")
	default:
		switch m.section {
		case youSectionInvites:
			return " " + helpEntry("j/k", "nav") + "  " + helpEntry("c", "copy link") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
		case youSectionLeaderboard:
			return " " + helpEntry("j/k", "nav") + "  " + helpEntry("f", "follow") + "  " + helpEntry("p", "peek") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
		default:
			return " " + helpEntry("j/k", "nav") + "  " + helpEntry("e", "edit") + "  " + helpEntry("a", "add") + "  " + helpEntry("d", "remove") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
		}
	}
}

func (m youModel) View() string {
	var sb strings.Builder

	// -- Identity section --
	if m.me != nil {
		// Emblem + login
		emblem := GuildEmblem(m.me.GuildID)
		nameStr := selectedStyle.Render(m.me.GitHubLogin)
		if emblem != "" {
			sb.WriteString(" " + emblem + " " + nameStr + "\n")
		} else {
			sb.WriteString(" " + nameStr + "\n")
		}

		// Guild · #N · Edition
		parts := []string{}
		if m.me.GuildID != "" {
			parts = append(parts, GuildStyle(m.me.GuildID).Render(m.me.GuildID))
		}
		if m.me.CardNumber > 0 {
			parts = append(parts, metaStyle.Render(fmt.Sprintf("#%d", m.me.CardNumber)))
		}
		if m.me.Edition != "" {
			parts = append(parts, metaStyle.Render(m.me.Edition))
		}
		if len(parts) > 0 {
			sb.WriteString("   " + strings.Join(parts, dimStyle.Render(" · ")) + "\n")
		}

		// Grimoire quip in italic gold
		quip := grimoireQuip(m.me)
		if quip != "" {
			sb.WriteString("   " + grimVoiceStyle.Render(quip) + "\n")
		}
	}

	if m.statusMsg != "" {
		sb.WriteString("\n " + upvoteStyle.Render(m.statusMsg) + "\n")
	}

	// -- Route to browse panel or unified view --
	if m.browsing {
		sb.WriteString(m.viewWorkshop())
	} else {
		sb.WriteString(m.viewWorkshopSection())
		sb.WriteString(m.viewInvitesSection())
		sb.WriteString(m.viewLeaderboardSection())
		sb.WriteString(" " + helpKeyStyle.Render("b") + " " + helpLabelStyle.Render("browse all magicians →") + "\n")
	}

	return sb.String()
}

// viewWorkshopSection renders the compact workshop project list for the unified view.
func (m youModel) viewWorkshopSection() string {
	var sb strings.Builder

	sb.WriteString("\n " + sectionHeaderStyle.Render(fmt.Sprintf("── WORKSHOP %d projects ──", len(m.projects))) + "\n")

	// Adding mode: two-field form
	if m.wsState == wsAdding {
		sb.WriteString(m.renderAddForm())
		return sb.String()
	}

	if len(m.projects) == 0 {
		sb.WriteString("   " + dimStyle.Render("no projects yet · press a to add one") + "\n")
		return sb.String()
	}

	maxRows := len(m.projects)
	if maxRows > 8 {
		maxRows = 8
	}

	for i := 0; i < maxRows; i++ {
		proj := m.projects[i]
		isActive := i == m.wsCursor && m.section == youSectionWorkshop

		cursor := "  "
		if isActive {
			cursor = accentStyle.Render("▸") + " "
		}

		nameStr := normalStyle.Render(truncStr(proj.Name, 24))
		if isActive {
			nameStr = selectedStyle.Render(truncStr(proj.Name, 24))
		}

		dateStr := metaStyle.Render(formatTime(proj.UpdatedAt))

		// Edit mode on selected row
		if i == m.wsCursor && m.wsState == wsEditing {
			fmt.Fprintf(&sb, " %s%s  %s\n", cursor, nameStr, dateStr)
			sb.WriteString("   " + inputPromptStyle.Render(">") + " " + m.wsEditText + accentStyle.Render("_") + "\n")
			continue
		}

		// Delete confirmation on selected row
		if i == m.wsCursor && m.wsState == wsDeleting {
			fmt.Fprintf(&sb, " %s%s  %s\n", cursor, nameStr, dateStr)
			sb.WriteString("   " + rejectStyle.Render("delete this project? ") +
				accentStyle.Render("y") + dimStyle.Render("/") + dimStyle.Render("n") + "\n")
			continue
		}

		insightStr := ""
		if proj.Insight != "" {
			insightStr = "\n     " + dimStyle.Render(proj.Insight)
		}

		fmt.Fprintf(&sb, " %s%s  %s%s\n", cursor, nameStr, dateStr, insightStr)
	}

	return sb.String()
}

// viewInvitesSection renders the invites section for the unified view.
func (m youModel) viewInvitesSection() string {
	if len(m.invites) == 0 {
		return ""
	}

	var sb strings.Builder
	var available []domain.Invite
	claimed := 0
	for _, inv := range m.invites {
		if inv.UsedBy == nil {
			available = append(available, inv)
		} else {
			claimed++
		}
	}
	sb.WriteString("\n " + sectionHeaderStyle.Render(fmt.Sprintf("── INVITES %d available ──", len(available))) + "\n")
	for i, inv := range available {
		isActive := i == m.inviteCursor && m.section == youSectionInvites
		cursor := "  "
		if isActive {
			cursor = accentStyle.Render("▸") + " "
		}
		sb.WriteString(" " + cursor + accentStyle.Render("grimora.ai/join/"+inv.Code) + "\n")
	}
	sb.WriteString("   " + metaStyle.Render(fmt.Sprintf("%d claimed · %d more forged spells until next invite", claimed, inviteSpellThreshold)) + "\n")

	return sb.String()
}

// viewLeaderboardSection renders the compact leaderboard for the unified view.
func (m youModel) viewLeaderboardSection() string {
	var sb strings.Builder

	sb.WriteString("\n " + sectionHeaderStyle.Render(fmt.Sprintf("── LEADERBOARD %d magicians ──", len(m.cards))) + "\n")

	if m.loading && len(m.cards) == 0 {
		sb.WriteString(" " + dimStyle.Render("loading...") + "\n")
		return sb.String()
	}
	if m.err != "" {
		sb.WriteString(" " + dimStyle.Render("error: "+m.err) + "\n")
		return sb.String()
	}

	maxRows := len(m.cards)
	if maxRows > 10 {
		maxRows = 10
	}

	for i := 0; i < maxRows; i++ {
		card := m.cards[i]
		rank := i + 1
		isActive := i == m.cursor && m.section == youSectionLeaderboard

		isYou := m.me != nil && card.GitHubLogin == m.me.GitHubLogin

		cursor := " "
		if isActive {
			cursor = accentStyle.Render("▸")
		}

		rankLabel := fmt.Sprintf("#%-3d", rank)
		rankStr := rankStyle(rank).Render(rankLabel)
		if isYou {
			rankStr = accentStyle.Render(rankLabel)
		}

		var loginStyled string
		if isYou {
			loginStyled = selectedStyle.Render(fmt.Sprintf("%-16s", "you"))
		} else {
			loginStyled = GuildStyle(card.GuildID).Render(fmt.Sprintf("%-16s", card.GitHubLogin))
		}

		spells := metaStyle.Render(fmt.Sprintf("%d spells", card.SpellCount))

		// Movement column
		moveStr := renderMove(card.Move)

		// Potency column
		potencyStr := ""
		if card.TotalPotency > 0 {
			potencyStr = potencyStyle(card.TotalPotency).Render(fmt.Sprintf("P%d", card.TotalPotency))
		}

		youMarker := ""
		if isYou {
			youMarker = " " + accentStyle.Render("<- you")
		}

		row := fmt.Sprintf(" %s %s  %s  %s", cursor, rankStr, loginStyled, spells)
		if potencyStr != "" {
			row += "  " + potencyStr
		}
		if moveStr != "" {
			row += "  " + moveStr
		}
		row += youMarker + "\n"
		sb.WriteString(row)
	}

	return sb.String()
}

func (m youModel) viewWorkshop() string {
	var sb strings.Builder

	sb.WriteString("\n " + sectionHeaderStyle.Render(fmt.Sprintf("── WORKSHOP %d projects ──", len(m.projects))) + "\n")

	// Adding mode: two-field form
	if m.wsState == wsAdding {
		sb.WriteString(m.renderAddForm())
		return sb.String()
	}

	if len(m.projects) == 0 && m.wsState != wsAdding {
		sb.WriteString("   " + dimStyle.Render("no projects yet · press a to add one") + "\n")
		return sb.String()
	}

	for i, proj := range m.projects {
		isSelected := i == m.wsCursor

		cursor := "  "
		if isSelected {
			cursor = accentStyle.Render("▸") + " "
		}

		nameStr := normalStyle.Render(truncStr(proj.Name, 24))
		if isSelected {
			nameStr = selectedStyle.Render(truncStr(proj.Name, 24))
		}

		dateStr := metaStyle.Render(formatTime(proj.UpdatedAt))

		// Edit mode on selected row
		if i == m.wsCursor && m.wsState == wsEditing {
			fmt.Fprintf(&sb, " %s%s  %s\n", cursor, nameStr, dateStr)
			sb.WriteString("   " + inputPromptStyle.Render(">") + " " + m.wsEditText + accentStyle.Render("_") + "\n")
			continue
		}

		// Delete confirmation on selected row
		if i == m.wsCursor && m.wsState == wsDeleting {
			fmt.Fprintf(&sb, " %s%s  %s\n", cursor, nameStr, dateStr)
			sb.WriteString("   " + rejectStyle.Render("delete this project? ") +
				accentStyle.Render("y") + dimStyle.Render("/") + dimStyle.Render("n") + "\n")
			continue
		}

		insightStr := ""
		if proj.Insight != "" {
			insightStr = "  " + dimStyle.Render(proj.Insight)
		}

		fmt.Fprintf(&sb, " %s%s  %s%s\n", cursor, nameStr, dateStr, insightStr)
	}

	// Browse panel: show leaderboard cards enriched with emblem + project info
	if len(m.cards) > 0 {
		sb.WriteString("\n " + sectionHeaderStyle.Render("── BROWSE ──") + "\n")
		maxBrowse := m.height - 20
		if maxBrowse < 3 {
			maxBrowse = 6
		}
		shown := len(m.cards)
		if shown > maxBrowse {
			shown = maxBrowse
		}
		for i := 0; i < shown; i++ {
			card := m.cards[i]
			emblem := card.Emblem
			if emblem == "" {
				emblem = GuildEmblem(card.GuildID)
			}

			nameColored := GuildStyle(card.GuildID).Render(card.GitHubLogin)
			details := []string{card.GuildID}
			details = append(details, fmt.Sprintf("%d spells", card.SpellCount))
			if card.TotalPotency > 0 {
				details = append(details, fmt.Sprintf("P%d", card.TotalPotency))
			}
			if card.City != "" {
				details = append(details, card.City)
			}
			detailStr := metaStyle.Render(strings.Join(details, " · "))

			line := "   "
			if emblem != "" {
				line += emblem + " "
			}
			line += nameColored + " " + detailStr + "\n"
			sb.WriteString(line)
		}
	}

	return sb.String()
}

func (m youModel) renderAddForm() string {
	var sb strings.Builder
	sb.WriteString("\n")

	nameCursor := ""
	insightCursor := ""
	if m.wsAddFocus == 0 {
		nameCursor = accentStyle.Render("_")
	} else {
		insightCursor = accentStyle.Render("_")
	}

	nameLabel := inputPromptStyle.Render("name:")
	insightLabel := inputPromptStyle.Render("insight:")

	var nameLine, insightLine string
	if m.wsAddFocus == 0 {
		nameLine = "   " + accentStyle.Render(">") + " " + nameLabel + " " + m.wsAddName + nameCursor
		insightLine = "     " + insightLabel + " " + dimStyle.Render(m.wsAddInsight)
	} else {
		nameLine = "     " + nameLabel + " " + dimStyle.Render(m.wsAddName)
		insightLine = "   " + accentStyle.Render(">") + " " + insightLabel + " " + m.wsAddInsight + insightCursor
	}

	sb.WriteString(nameLine + "\n")
	sb.WriteString(insightLine + "\n")
	return sb.String()
}

// grimoireQuip returns a short italic-gold flavor line based on the magician's profile.
// It uses archetype when available and falls back to a generic line.
func grimoireQuip(me *domain.Magician) string {
	if me == nil {
		return ""
	}
	archetypeQuips := map[string]string{
		"architect":  "you design the spells that others dare not imagine.",
		"alchemist":  "you transmute raw thought into potent incantations.",
		"oracle":     "the patterns speak to you before others notice them.",
		"tinkerer":   "every system bends under your curious hands.",
		"sentinel":   "you guard the craft with precision and care.",
		"chronicler": "your words outlast the moment they were cast.",
	}
	if me.Archetype != "" {
		if q, ok := archetypeQuips[me.Archetype]; ok {
			return q
		}
	}
	return "your spells shape the realm."
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

// renderMove formats a leaderboard rank movement value as colored text.
// Positive values are rendered green (↑N), negative red (↓N), zero dim (–).
func renderMove(move int) string {
	switch {
	case move > 0:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#4ade80")).Render(fmt.Sprintf("↑%d", move))
	case move < 0:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#b45555")).Render(fmt.Sprintf("↓%d", -move))
	default:
		return dimStyle.Render("–")
	}
}
