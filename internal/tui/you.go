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
	client     *client.Client
	invites    []domain.Invite
	me         *domain.Magician
	forgeStats *domain.ForgeStats
	statusMsg  string
	width      int
	height     int
	section    youSection // active section for navigation

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
		m.forgeStats = msg.stats
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

	sb.WriteString(m.viewStatsBar())
	sb.WriteString(m.viewBuildJournal())
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

// viewStatsBar renders the craft stats summary between identity and build journal.
func (m youModel) viewStatsBar() string {
	var sb strings.Builder

	w := m.width - 4
	if w < 20 {
		w = 46
	}
	sb.WriteString("\n " + sectionHeaderStyle.Render("── CRAFT "+strings.Repeat("─", max(w-9, 1))) + "\n")

	// Count shipped vs building from projectUpdates
	shipped := 0
	building := 0
	for _, proj := range m.projects {
		if updates, ok := m.projectUpdates[proj.ID.String()]; ok {
			if projectStatus(updates) == "shipped" {
				shipped++
			} else {
				building++
			}
		} else {
			building++ // no updates loaded yet = assume building
		}
	}

	parts := []string{
		accentStyle.Render(fmt.Sprintf("%d", shipped)) + dimStyle.Render(" shipped"),
		dimStyle.Render(fmt.Sprintf("%d", building)) + dimStyle.Render(" building"),
	}

	if m.forgeStats != nil {
		parts = append(parts,
			dimStyle.Render(fmt.Sprintf("%d", m.forgeStats.SpellsForged))+" "+dimStyle.Render("spells"),
			goldStyle.Render(fmt.Sprintf("P%d", m.forgeStats.TotalPotency)),
			accentStyle.Render(fmt.Sprintf("#%d", m.forgeStats.Rank)),
		)
	}

	sb.WriteString("   " + strings.Join(parts, dimStyle.Render("   ")) + "\n")
	return sb.String()
}

// viewBuildJournal renders the build journal with vertical timelines per project.
func (m youModel) viewBuildJournal() string {
	var sb strings.Builder

	sb.WriteString("\n " + sectionHeaderStyle.Render(fmt.Sprintf("── BUILD JOURNAL %d projects ──", len(m.projects))) + "\n")

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
		updates := m.projectUpdates[proj.ID.String()]

		sb.WriteString(m.renderProjectCard(proj, updates, isActive, i))
	}

	return sb.String()
}

// renderProjectCard renders a single project with its vertical timeline.
func (m youModel) renderProjectCard(proj domain.WorkshopProject, updates []domain.ProjectUpdate, isActive bool, idx int) string {
	var sb strings.Builder

	// Project header line: cursor + name + badge
	cursor := "  "
	if isActive {
		cursor = accentStyle.Render("▸") + " "
	}

	nameStr := normalStyle.Render(truncStr(proj.Name, 30))
	if isActive {
		nameStr = selectedStyle.Render(truncStr(proj.Name, 30))
	}

	status := projectStatus(updates)
	badge := ""
	if status == "shipped" {
		badge = goldStyle.Render("shipped")
	} else {
		badge = dimStyle.Render("building")
	}

	// Right-align badge
	nameWidth := lipgloss.Width(cursor + truncStr(proj.Name, 30))
	padLen := m.width - 2 - nameWidth - lipgloss.Width(badge)
	if padLen < 2 {
		padLen = 2
	}
	pad := strings.Repeat(" ", padLen)

	fmt.Fprintf(&sb, " %s%s%s%s\n", cursor, nameStr, pad, badge)

	// Insight line
	if proj.Insight != "" {
		sb.WriteString("   " + dimStyle.Render(proj.Insight) + "\n")
	}

	// Delete confirmation overlay
	if idx == m.wsCursor && m.wsState == wsDeleting {
		sb.WriteString("   " + rejectStyle.Render("delete this project? ") +
			accentStyle.Render("y") + dimStyle.Render("/") + dimStyle.Render("n") + "\n")
		return sb.String()
	}

	// Timeline
	if len(updates) > 0 {
		sb.WriteString("   " + dimStyle.Render("│") + "\n")

		// Show max 5 most recent; if more, hint at earlier
		maxShow := 5
		start := 0
		if len(updates) > maxShow {
			start = len(updates) - maxShow
			sb.WriteString("   " + dimStyle.Render(fmt.Sprintf("│ ... %d earlier updates", start)) + "\n")
		}

		for j := start; j < len(updates); j++ {
			u := updates[j]
			ts := metaStyle.Render(formatTime(u.CreatedAt))

			// Right-align timestamp
			var dotLine string
			switch u.Kind {
			case "start":
				dotLine = "   " + accentStyle.Render("●") + " " + accentStyle.Render("started building")
			case "ship":
				dotLine = "   " + goldStyle.Render("✦") + " " + goldStyle.Render("shipped")
			default:
				dotLine = "   " + dimStyle.Render("●") + " " + dimStyle.Render(truncStr(u.Body, 40))
			}

			tsPad := m.width - 2 - lipgloss.Width(dotLine) - lipgloss.Width(ts)
			if tsPad < 2 {
				tsPad = 2
			}
			sb.WriteString(dotLine + strings.Repeat(" ", tsPad) + ts + "\n")

			// Body text on connector line (for start/ship with body)
			if u.Kind == "ship" && u.Body != "" {
				sb.WriteString("   " + dimStyle.Render("│") + " " + dimStyle.Render(u.Body) + "\n")
			} else if u.Kind == "start" && u.Body != "" {
				sb.WriteString("   " + dimStyle.Render("│") + " " + dimStyle.Render(u.Body) + "\n")
			}

			// Connector between dots (except after last)
			if j < len(updates)-1 {
				sb.WriteString("   " + dimStyle.Render("│") + "\n")
			}
		}

		// Active projects end with continuation marker
		if status != "shipped" {
			sb.WriteString("\n   " + dimStyle.Render("○ ···") + "\n")
		}
	}

	sb.WriteString("\n")
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
