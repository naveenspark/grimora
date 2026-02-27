package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

// workshopState is the state machine for workshop CRUD interactions.
type workshopState int

const (
	wsNormal   workshopState = iota
	wsEditing                // editing insight of selected project
	wsAdding                 // adding new project (name + insight fields)
	wsDeleting               // delete confirmation
)

// -- messages --

type youInvitesLoadedMsg struct {
	invites []domain.Invite
	err     error
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

type projectUpdatesMsg struct {
	projectID string
	updates   []domain.ProjectUpdate
	err       error
}

// -- model --

// youSection identifies which section the cursor is in.
type youSection int

const (
	youSectionWorkshop youSection = iota
	youSectionInvites
)

// inviteSpellThreshold is the number of forged spells required for the next invite code.
const inviteSpellThreshold = 10

type youModel struct {
	client    *client.Client
	invites   []domain.Invite
	me        *domain.Magician
	statusMsg string
	width     int
	height    int
	section   youSection // active section for navigation

	// workshop
	projects       []domain.WorkshopProject
	projectUpdates map[string][]domain.ProjectUpdate
	wsCursor       int
	wsState        workshopState
	wsAddName      string // name field when adding/editing
	wsAddInsight   string // insight field when adding/editing
	wsAddFocus     int    // 0=name, 1=insight

	// invites
	inviteCursor int
}

func newYouModel(c *client.Client) youModel {
	return youModel{client: c, projectUpdates: make(map[string][]domain.ProjectUpdate)}
}

func (m youModel) Init() tea.Cmd {
	return tea.Batch(m.loadInvites(), m.loadWorkshop())
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

func (m youModel) loadProjectUpdates(projectID string) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		updates, err := c.ListProjectUpdates(context.Background(), projectID)
		return projectUpdatesMsg{projectID: projectID, updates: updates, err: err}
	}
}

func (m youModel) Update(msg tea.Msg) (youModel, tea.Cmd) {
	switch msg := msg.(type) {
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
		// Load project updates for status badges
		var cmds []tea.Cmd
		if m.client != nil {
			for _, p := range m.projects {
				cmds = append(cmds, m.loadProjectUpdates(p.ID.String()))
			}
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case projectUpdatesMsg:
		if msg.err == nil {
			m.projectUpdates[msg.projectID] = msg.updates
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
		m.wsAddName = ""
		m.wsAddInsight = ""
		m.wsAddFocus = 0
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
		m.navDown()

	case "k", "up":
		m.navUp()

	case "e":
		// Edit selected workshop project (name + insight)
		if m.section == youSectionWorkshop && len(m.projects) > 0 && m.wsCursor < len(m.projects) {
			m.wsState = wsEditing
			m.wsAddName = m.projects[m.wsCursor].Name
			m.wsAddInsight = m.projects[m.wsCursor].Insight
			m.wsAddFocus = 0
		}

	case "a":
		// Add new workshop project
		m.wsState = wsAdding
		m.wsAddName = ""
		m.wsAddInsight = ""
		m.wsAddFocus = 0

	case "d":
		// Delete selected workshop project
		if m.section == youSectionWorkshop && len(m.projects) > 0 && m.wsCursor < len(m.projects) {
			m.wsState = wsDeleting
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
		return m, tea.Batch(m.loadInvites(), m.loadWorkshop())
	}
	return m, nil
}

func (m youModel) handleKeyEditing(msg tea.KeyMsg) (youModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.wsAddFocus = 1 - m.wsAddFocus
	case "enter":
		name := strings.TrimSpace(m.wsAddName)
		if name == "" {
			m.statusMsg = "name required"
			return m, nil
		}
		if len(m.projects) > 0 && m.wsCursor < len(m.projects) {
			id := m.projects[m.wsCursor].ID.String()
			insight := strings.TrimSpace(m.wsAddInsight)
			m.projects[m.wsCursor].Name = name
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

func (m youModel) handleKeyAdding(msg tea.KeyMsg) (youModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.wsAddFocus = 1 - m.wsAddFocus
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
		}
	case youSectionInvites:
		avail := m.availableInvites()
		if m.inviteCursor < len(avail)-1 {
			m.inviteCursor++
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
	}
}

// helpKeys returns context-sensitive help text based on the current state.
func (m youModel) helpKeys() string {
	switch m.wsState {
	case wsEditing, wsAdding:
		return helpEntry("tab", "next") + "  " + helpEntry("enter", "save") + "  " + helpEntry("esc", "cancel")
	case wsDeleting:
		return helpEntry("y", "confirm") + "  " + helpEntry("n", "cancel")
	default:
		switch m.section {
		case youSectionInvites:
			return helpEntry("j/k", "nav") + "  " + helpEntry("c", "copy link") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
		default:
			return helpEntry("j/k", "nav") + "  " + helpEntry("e", "edit") + "  " + helpEntry("a", "add") + "  " + helpEntry("d", "remove") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
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

	sb.WriteString(m.viewWorkshopSection())
	sb.WriteString(m.viewInvitesSection())

	return sb.String()
}

// projectStatus returns "shipped" if any update has kind "ship", else "building".
func projectStatus(updates []domain.ProjectUpdate) string {
	for _, u := range updates {
		if u.Kind == "ship" {
			return "shipped"
		}
	}
	return "building"
}

// viewWorkshopSection renders the compact workshop project list.
func (m youModel) viewWorkshopSection() string {
	var sb strings.Builder

	sb.WriteString("\n " + sectionHeaderStyle.Render(fmt.Sprintf("── WORKSHOP %d projects ──", len(m.projects))) + "\n")

	// Adding/editing mode: two-field form
	if m.wsState == wsAdding || m.wsState == wsEditing {
		sb.WriteString(m.renderEditForm())
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

		badge := ""
		if updates, ok := m.projectUpdates[proj.ID.String()]; ok {
			status := projectStatus(updates)
			if status == "shipped" {
				badge = " " + accentStyle.Render("[shipped]")
			} else {
				badge = " " + dimStyle.Render("[building]")
			}
		}

		// Delete confirmation on selected row
		if i == m.wsCursor && m.wsState == wsDeleting {
			fmt.Fprintf(&sb, " %s%s  %s%s\n", cursor, nameStr, dateStr, badge)
			sb.WriteString("   " + rejectStyle.Render("delete this project? ") +
				accentStyle.Render("y") + dimStyle.Render("/") + dimStyle.Render("n") + "\n")
			continue
		}

		insightStr := ""
		if proj.Insight != "" {
			insightStr = "\n     " + dimStyle.Render(proj.Insight)
		}

		fmt.Fprintf(&sb, " %s%s  %s%s%s\n", cursor, nameStr, dateStr, badge, insightStr)
	}

	return sb.String()
}

// viewInvitesSection renders the invites section.
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

func (m youModel) renderEditForm() string {
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
	sb.WriteString("   " + dimStyle.Render("tab next · enter save · esc cancel") + "\n")
	return sb.String()
}

// grimoireQuip returns a short italic-gold flavor line based on the magician's profile.
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
