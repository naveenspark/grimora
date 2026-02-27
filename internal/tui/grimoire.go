package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

type grimoireMode int

const (
	grimoireModeSpells grimoireMode = iota
	grimoireModeWeapons
)

type grimoireModel struct {
	client    *client.Client
	mode      grimoireMode
	spells    []domain.Spell
	weapons   []domain.Weapon
	cursor    int
	search    string
	editing   bool // true when typing in search
	tagFilter string
	sortBy    string // "new", "top", or "casts"
	detail    bool   // in detail view
	err       error
	width     int
	height    int
	loading   bool
	statusMsg string
}

// Reuse message types from old spells/weapons
type spellsLoadedMsg struct {
	spells []domain.Spell
	err    error
}

type weaponsLoadedMsg struct {
	weapons []domain.Weapon
	err     error
}

type upvoteResultMsg struct{ err error }
type copyResultMsg struct{ err error }
type saveWeaponResultMsg struct{ err error }

func newGrimoireModel(c *client.Client) grimoireModel {
	return grimoireModel{
		client:  c,
		loading: true,
		sortBy:  "new",
	}
}

func (m grimoireModel) loadSpells() tea.Cmd {
	return func() tea.Msg {
		var spells []domain.Spell
		var err error
		if m.search != "" {
			spells, err = m.client.SearchSpells(context.Background(), m.search)
		} else {
			spells, err = m.client.ListSpells(context.Background(), m.tagFilter, m.sortBy, pageSize, 0)
		}
		return spellsLoadedMsg{spells: spells, err: err}
	}
}

func (m grimoireModel) loadWeapons() tea.Cmd {
	return func() tea.Msg {
		var weapons []domain.Weapon
		var err error
		if m.search != "" {
			weapons, err = m.client.SearchWeapons(context.Background(), m.search)
		} else {
			weapons, err = m.client.ListWeapons(context.Background(), pageSize, 0)
		}
		return weaponsLoadedMsg{weapons: weapons, err: err}
	}
}

func (m grimoireModel) loadCurrent() tea.Cmd {
	if m.mode == grimoireModeWeapons {
		return m.loadWeapons()
	}
	return m.loadSpells()
}

func (m grimoireModel) Init() tea.Cmd {
	return m.loadCurrent()
}

func (m grimoireModel) Update(msg tea.Msg) (grimoireModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spellsLoadedMsg:
		m.loading = false
		m.spells = msg.spells
		m.err = msg.err
		if m.cursor >= len(m.spells) {
			m.cursor = 0
		}
		return m, nil

	case weaponsLoadedMsg:
		m.loading = false
		m.weapons = msg.weapons
		m.err = msg.err
		if m.cursor >= len(m.weapons) {
			m.cursor = 0
		}
		return m, nil

	case upvoteResultMsg:
		if msg.err != nil {
			if strings.Contains(msg.err.Error(), "not authenticated") {
				m.statusMsg = "not authenticated -- run: grimora login"
			} else {
				m.statusMsg = fmt.Sprintf("upvote failed: %v", msg.err)
			}
			return m, nil
		}
		m.statusMsg = "upvoted!"
		return m, m.loadSpells()

	case saveWeaponResultMsg:
		if msg.err != nil {
			if strings.Contains(msg.err.Error(), "not authenticated") {
				m.statusMsg = "not authenticated -- run: grimora login"
			} else {
				m.statusMsg = fmt.Sprintf("save failed: %v", msg.err)
			}
		} else {
			m.statusMsg = "saved!"
		}
		return m, m.loadWeapons()

	case copyResultMsg:
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
		if m.editing {
			return m.updateSearch(msg)
		}
		if m.detail {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m grimoireModel) updateSearch(msg tea.KeyMsg) (grimoireModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.editing = false
		m.loading = true
		return m, m.loadCurrent()
	case "esc":
		m.editing = false
		m.search = ""
		m.loading = true
		return m, m.loadCurrent()
	default:
		m.search = editRune(m.search, msg.String())
	}
	return m, nil
}

func (m grimoireModel) updateList(msg tea.KeyMsg) (grimoireModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		max := m.listLen() - 1
		if m.cursor < max {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if m.listLen() > 0 {
			m.detail = true
		}
	case "/":
		m.editing = true
		m.search = ""
	case "w":
		// Toggle spells/weapons
		if m.mode == grimoireModeSpells {
			m.mode = grimoireModeWeapons
		} else {
			m.mode = grimoireModeSpells
		}
		m.cursor = 0
		m.detail = false
		m.loading = true
		return m, m.loadCurrent()
	case "s":
		if m.mode == grimoireModeSpells {
			switch m.sortBy {
			case "new":
				m.sortBy = "top"
			case "top":
				m.sortBy = "casts"
			default:
				m.sortBy = "new"
			}
			m.cursor = 0
			m.loading = true
			return m, m.loadSpells()
		}
		// For weapons, s = save
		if m.mode == grimoireModeWeapons && m.cursor < len(m.weapons) {
			weapon := m.weapons[m.cursor]
			return m, func() tea.Msg {
				err := m.client.SaveWeapon(context.Background(), weapon.ID.String())
				return saveWeaponResultMsg{err: err}
			}
		}
	case "t":
		if m.mode == grimoireModeSpells {
			// Cycle through display tags (no filter → first tag → ... → last tag → no filter)
			if m.tagFilter == "" {
				m.tagFilter = displayTags[0]
			} else {
				found := false
				for i, tag := range displayTags {
					if tag == m.tagFilter {
						if i+1 < len(displayTags) {
							m.tagFilter = displayTags[i+1]
						} else {
							m.tagFilter = "" // wrap to "all"
						}
						found = true
						break
					}
				}
				if !found {
					m.tagFilter = ""
				}
			}
			m.cursor = 0
			m.loading = true
			return m, m.loadSpells()
		}
	case "u":
		if m.mode == grimoireModeSpells && m.cursor < len(m.spells) {
			spell := m.spells[m.cursor]
			return m, func() tea.Msg {
				err := m.client.UpvoteSpell(context.Background(), spell.ID.String())
				return upvoteResultMsg{err: err}
			}
		}
	case "c":
		if m.mode == grimoireModeSpells && m.cursor < len(m.spells) {
			text := m.spells[m.cursor].Text
			return m, func() tea.Msg {
				err := clipboard.WriteAll(text)
				return copyResultMsg{err: err}
			}
		}
	case "r":
		m.loading = true
		return m, m.loadCurrent()
	}
	return m, nil
}

func (m grimoireModel) updateDetail(msg tea.KeyMsg) (grimoireModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.detail = false
	case "u":
		if m.mode == grimoireModeSpells && m.cursor < len(m.spells) {
			spell := m.spells[m.cursor]
			return m, func() tea.Msg {
				err := m.client.UpvoteSpell(context.Background(), spell.ID.String())
				return upvoteResultMsg{err: err}
			}
		}
	case "c":
		if m.mode == grimoireModeSpells && m.cursor < len(m.spells) {
			text := m.spells[m.cursor].Text
			return m, func() tea.Msg {
				err := clipboard.WriteAll(text)
				return copyResultMsg{err: err}
			}
		}
	case "s":
		if m.mode == grimoireModeWeapons && m.cursor < len(m.weapons) {
			weapon := m.weapons[m.cursor]
			return m, func() tea.Msg {
				err := m.client.SaveWeapon(context.Background(), weapon.ID.String())
				return saveWeaponResultMsg{err: err}
			}
		}
	case "p":
		if m.mode == grimoireModeSpells && m.cursor < len(m.spells) {
			spell := m.spells[m.cursor]
			if spell.Author != nil {
				login := spell.Author.Login
				return m, func() tea.Msg {
					return showPeekMsg{login: login}
				}
			}
		}
	}
	return m, nil
}

func (m grimoireModel) listLen() int {
	if m.mode == grimoireModeWeapons {
		return len(m.weapons)
	}
	return len(m.spells)
}

// displayTags is the curated set shown in the inline tag bar (matches mockup).
var displayTags = []string{
	"debugging", "database", "performance", "architecture",
	"ai-prompts", "testing", "code-review", "security", "observability",
}

func (m grimoireModel) View() string {
	// Detail view
	if m.detail {
		if m.mode == grimoireModeWeapons {
			return m.viewWeaponDetail()
		}
		return m.viewSpellDetail()
	}

	// List view
	var b strings.Builder

	// Header line (hide tagline at narrow widths)
	if m.width >= 50 {
		b.WriteString(" " + grimLabelStyle.Render("THE GRIMOIRE") + "  " + grimVoiceStyle.Render("Knowledge is a shared weapon.") + "\n")
	} else {
		b.WriteString(" " + grimLabelStyle.Render("THE GRIMOIRE") + "\n")
	}

	// --- Search + mode toggle ---
	if m.editing {
		b.WriteString(" " + searchStyle.Render("/ "+m.search+"\u2588"))
	} else if m.search != "" {
		b.WriteString(" " + searchStyle.Render("/ "+m.search))
	} else {
		b.WriteString(" " + dimStyle.Render("/ search..."))
	}

	// Mode toggle: [spells] [weapons]
	b.WriteString("   ")
	if m.mode == grimoireModeSpells {
		b.WriteString(searchStyle.Render("[spells]"))
		b.WriteString(" ")
		b.WriteString(dimStyle.Render("[weapons]"))
	} else {
		b.WriteString(dimStyle.Render("[spells]"))
		b.WriteString(" ")
		b.WriteString(searchStyle.Render("[weapons]"))
	}
	b.WriteString("  " + helpKeyStyle.Render("w"))
	b.WriteString("\n")

	// --- Tag bar + sort (spells mode only) ---
	if m.mode == grimoireModeSpells {
		// Sort indicator at the end: "new↑ s" (~8 chars)
		sortLabel := m.sortBy + "\u2191"
		sortPart := "   " + searchStyle.Render(sortLabel) + " " + helpKeyStyle.Render("s")
		sortWidth := lipgloss.Width(sortPart)

		b.WriteString(" ")
		usedWidth := 1 // leading space
		for i, tag := range displayTags {
			sep := "  "
			if i == 0 {
				sep = ""
			}
			needed := len(sep) + len(tag)
			if usedWidth+needed+sortWidth > m.width {
				break // don't overflow
			}
			b.WriteString(sep)
			isActive := tag == m.tagFilter
			if isActive {
				b.WriteString(TagStyle(tag).Bold(true).Render(tag))
			} else {
				b.WriteString(dimStyle.Render(tag))
			}
			usedWidth += needed
		}
		b.WriteString(sortPart)
		b.WriteString("\n")
	}

	// Separator
	sepW := m.width - 2
	if sepW < 4 {
		sepW = 4
	}
	b.WriteString(" " + metaStyle.Render(strings.Repeat("\u2500", sepW)) + "\n")

	if m.statusMsg != "" {
		b.WriteString(" " + upvoteStyle.Render(m.statusMsg) + "\n")
	}

	if m.loading {
		b.WriteString(" " + dimStyle.Render("loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(" " + dimStyle.Render(fmt.Sprintf("error: %v", m.err)))
		return b.String()
	}

	if m.mode == grimoireModeWeapons {
		return b.String() + m.viewWeaponList()
	}
	return b.String() + m.viewSpellList()
}

func (m grimoireModel) viewSpellList() string {
	if len(m.spells) == 0 {
		return " " + dimStyle.Render("no spells found")
	}

	var b strings.Builder

	viewChrome := 10 // editorial + search/mode + tag bar + separator + detail chrome
	available := m.height - viewChrome
	if available < 6 {
		available = 6
	}
	maxVisible := available * 2 / 5
	if maxVisible < 3 {
		maxVisible = 3
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	for i := start; i < len(m.spells) && i < start+maxVisible; i++ {
		spell := m.spells[i]

		// Cursor indicator (▸ or space)
		cursor := "  "
		titleStyle := dimStyle
		if i == m.cursor {
			cursor = accentStyle.Render("▸") + " "
			titleStyle = normalStyle.Bold(true)
		}

		// Dot in tag color
		dot := TagStyle(spell.Tag).Render("●") + " "

		// Right-side columns: responsive based on width.
		// Wide (>=70): author(12) + casts(11) + potency(3) + gaps(4) = 30
		// Medium (>=45): casts(11) + potency(3) + gap(2) = 16
		// Narrow (<45): casts only(8) + gap(1) = 9
		showAuthor := m.width >= 70
		compactCasts := m.width < 45

		var rightParts []string
		rightWidth := 0
		if showAuthor {
			authorCol := ""
			if spell.Author != nil {
				name := spell.Author.Login
				if spell.Author.DisplayName != "" {
					name = spell.Author.DisplayName
				}
				if len(name) > 12 {
					name = name[:11] + "…"
				}
				authorCol = GuildStyle(spell.Author.GuildID).Render(fmt.Sprintf("%-12s", name))
			} else {
				authorCol = strings.Repeat(" ", 12)
			}
			rightParts = append(rightParts, authorCol)
			rightWidth += 13 // 12 + gap
		}
		if compactCasts {
			rightParts = append(rightParts, metaStyle.Render(fmt.Sprintf("%d", spell.Upvotes)+"c"))
			rightWidth += 5
		} else {
			rightParts = append(rightParts, metaStyle.Render(fmt.Sprintf("%6d casts", spell.Upvotes)))
			rightWidth += 12
		}
		if spell.Potency > 0 {
			rightParts = append(rightParts, potencyStyle(spell.Potency).Render(fmt.Sprintf("P%d", spell.Potency)))
			rightWidth += 4
		}

		// Title fills remaining space
		titleWidth := m.width - 4 - rightWidth // 4 = cursor(2) + dot(2)
		if titleWidth < 10 {
			titleWidth = 10
		}
		title := strings.ReplaceAll(spell.Text, "\n", " ")
		title = truncStr(title, titleWidth)
		titlePadded := fmt.Sprintf("%-*s", titleWidth, `"`+title+`"`)

		line := cursor + dot + titleStyle.Render(titlePadded) + " " + strings.Join(rightParts, " ")
		if i == m.cursor {
			padded := line + strings.Repeat(" ", max(m.width-lipgloss.Width(line), 0))
			b.WriteString(selectedRowBg.Render(padded) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	// Detail preview for selected spell (bottom portion)
	if m.cursor < len(m.spells) {
		spell := m.spells[m.cursor]
		b.WriteString("\n")

		header := " " + TagStyle(spell.Tag).Render("["+spell.Tag+"]")
		if spell.Model != "" {
			header += "  " + metaStyle.Render(spell.Model)
		}
		if spell.Upvotes > 0 {
			header += "  " + upvoteStyle.Render(fmt.Sprintf("^%d", spell.Upvotes))
		}
		if spell.Potency > 0 {
			header += "  " + potencyStyle(spell.Potency).Render(fmt.Sprintf("P%d", spell.Potency))
		}
		b.WriteString(header + "\n")

		detailWidth := m.width - 4
		if detailWidth < 40 {
			detailWidth = 40
		}
		maxDetailLines := available - maxVisible - 2
		if maxDetailLines < 2 {
			maxDetailLines = 2
		}
		wrapped := lipgloss.NewStyle().Width(detailWidth).Render(spell.Text)
		lines := strings.Split(wrapped, "\n")
		truncated := false
		if len(lines) > maxDetailLines {
			lines = lines[:maxDetailLines]
			truncated = true
		}
		for _, line := range lines {
			b.WriteString(" " + normalStyle.Render(line) + "\n")
		}
		if truncated {
			remaining := len(strings.Split(wrapped, "\n")) - maxDetailLines
			b.WriteString(" " + metaStyle.Render(fmt.Sprintf("\u2026 %d more lines (c to copy)", remaining)) + "\n")
		}
	}

	return truncateToHeight(b.String(), m.height)
}

func (m grimoireModel) viewWeaponList() string {
	if len(m.weapons) == 0 {
		return " " + dimStyle.Render("no weapons found")
	}

	var b strings.Builder

	viewChrome := 9 // editorial + search/mode + separator + detail chrome
	available := m.height - viewChrome
	if available < 6 {
		available = 6
	}
	maxVisible := available * 2 / 5
	if maxVisible < 3 {
		maxVisible = 3
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	for i := start; i < len(m.weapons) && i < start+maxVisible; i++ {
		w := m.weapons[i]

		cursor := "  "
		titleStyle := dimStyle
		if i == m.cursor {
			cursor = accentStyle.Render("▸") + " "
			titleStyle = normalStyle.Bold(true)
		}

		// Dot in category color
		dot := TagStyle(w.GitHubLanguage).Render("●") + " "

		// Right columns: category (10), stars (8)
		catCol := ""
		if w.GitHubLanguage != "" {
			catCol = TagStyle(w.GitHubLanguage).Render(fmt.Sprintf("%-10s", w.GitHubLanguage))
		} else {
			catCol = strings.Repeat(" ", 10)
		}
		starCol := upvoteStyle.Render(fmt.Sprintf("★%s", formatNum(w.GitHubStars)))

		// Title fills remaining space
		rightWidth := 10 + 8 + 3 // cat + stars + gaps
		titleWidth := m.width - 4 - rightWidth
		if titleWidth < 20 {
			titleWidth = 20
		}
		name := w.Name
		name = truncStr(name, titleWidth)
		namePadded := fmt.Sprintf("%-*s", titleWidth, name)

		line := cursor + dot + titleStyle.Render(namePadded) + " " + catCol + " " + starCol
		if i == m.cursor {
			padded := line + strings.Repeat(" ", max(m.width-lipgloss.Width(line), 0))
			b.WriteString(selectedRowBg.Render(padded) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	// Detail preview
	if m.cursor < len(m.weapons) {
		w := m.weapons[m.cursor]
		b.WriteString("\n")

		header := " " + selectedStyle.Render(w.Name)
		if w.GitHubLanguage != "" {
			header += "  " + TagStyle(w.GitHubLanguage).Render(w.GitHubLanguage)
		}
		if w.License != "" {
			header += "  " + metaStyle.Render(w.License)
		}
		b.WriteString(header + "\n")

		if w.Description != "" {
			detailWidth := m.width - 4
			if detailWidth < 30 {
				detailWidth = 60
			}
			maxDetailLines := available - maxVisible - 2
			if maxDetailLines < 2 {
				maxDetailLines = 2
			}
			wrapped := lipgloss.NewStyle().Width(detailWidth).Render(w.Description)
			lines := strings.Split(wrapped, "\n")
			if len(lines) > maxDetailLines {
				lines = lines[:maxDetailLines]
			}
			for _, line := range lines {
				b.WriteString(" " + normalStyle.Render(line) + "\n")
			}
		}

		stats := "\u2605 " + formatNum(w.GitHubStars) + "  forks " + formatNum(w.GitHubForks)
		if w.SaveCount > 0 {
			stats += fmt.Sprintf("  saves %d", w.SaveCount)
		}
		b.WriteString(" " + metaStyle.Render(stats) + "\n")
		b.WriteString(" " + metaStyle.Render(w.RepositoryURL) + "\n")
	}

	return truncateToHeight(b.String(), m.height)
}

func (m grimoireModel) viewSpellDetail() string {
	if m.cursor >= len(m.spells) {
		return ""
	}
	spell := m.spells[m.cursor]

	var b strings.Builder
	b.WriteString(" " + dimStyle.Render("<- back (esc)") + "\n")
	b.WriteString(" " + selectedStyle.Render(`"`+truncStr(spell.Text, 60)+`"`) + "\n")

	// Meta line: author · tag · PN · N casts · ^N
	meta := " "
	if spell.Author != nil {
		authorName := spell.Author.Login
		if spell.Author.DisplayName != "" {
			authorName = spell.Author.DisplayName
		}
		meta += GuildStyle(spell.Author.GuildID).Render(authorName) + metaStyle.Render(" · ")
	}
	meta += TagStyle(spell.Tag).Render(spell.Tag)
	if spell.Potency > 0 {
		meta += metaStyle.Render(" · ") + potencyStyle(spell.Potency).Render(fmt.Sprintf("P%d", spell.Potency))
	}
	if spell.Upvotes > 0 {
		meta += metaStyle.Render(fmt.Sprintf(" · %d casts", spell.Upvotes))
		meta += metaStyle.Render(fmt.Sprintf(" · \u2191%d", spell.Upvotes))
	}
	b.WriteString(meta + "\n")

	b.WriteString("\n")
	detailWidth := m.width - 4
	if detailWidth < 40 {
		detailWidth = 40
	}
	wrapped := lipgloss.NewStyle().Width(detailWidth).Render(spell.Text)
	for _, line := range strings.Split(wrapped, "\n") {
		b.WriteString(" " + normalStyle.Render(line) + "\n")
	}

	if len(spell.Stack) > 0 {
		b.WriteString("\n " + metaStyle.Render("stack: "+strings.Join(spell.Stack, ", ")) + "\n")
	}
	if spell.Context != "" {
		b.WriteString(" " + metaStyle.Render("context: "+spell.Context) + "\n")
	}

	// Grimoire voice block
	if spell.Voice != "" {
		b.WriteString("\n")
		b.WriteString(goldStyle.Render("\u2502") + " " + grimLabelStyle.Render("THE GRIMOIRE") + "\n")
		voiceWidth := detailWidth - 2
		if voiceWidth < 20 {
			voiceWidth = 20
		}
		voiceWrapped := lipgloss.NewStyle().Width(voiceWidth).Render(spell.Voice)
		for _, line := range strings.Split(voiceWrapped, "\n") {
			b.WriteString(goldStyle.Render("\u2502") + " " + grimVoiceStyle.Render(line) + "\n")
		}
	}

	// Comments section
	if len(spell.Comments) > 0 {
		b.WriteString("\n")
		b.WriteString(" " + sectionHeaderStyle.Render(fmt.Sprintf("COMMENTS (%d)", len(spell.Comments))) + "\n")
		for _, c := range spell.Comments {
			who := GuildStyle(c.GuildID).Render(c.Login)
			text := commentTextStyle.Render(c.Text)
			when := commentTimeStyle.Render(formatCommentTime(c.CreatedAt))
			fmt.Fprintf(&b, " %s  %s  %s\n", who, text, when)
		}
	}

	if m.statusMsg != "" {
		b.WriteString("\n " + upvoteStyle.Render(m.statusMsg) + "\n")
	}

	return truncateToHeight(b.String(), m.height)
}

func (m grimoireModel) viewWeaponDetail() string {
	if m.cursor >= len(m.weapons) {
		return ""
	}
	w := m.weapons[m.cursor]

	var b strings.Builder
	b.WriteString(" " + dimStyle.Render("<- back (esc)") + "\n")
	b.WriteString(" " + selectedStyle.Render(w.Name) + "\n")

	info := ""
	if w.GitHubLanguage != "" {
		info += " " + TagStyle(w.GitHubLanguage).Render(w.GitHubLanguage)
	}
	if w.License != "" {
		info += "  " + metaStyle.Render(w.License)
	}
	info += "  " + upvoteStyle.Render("\u2605"+formatNum(w.GitHubStars))
	b.WriteString(info + "\n\n")

	if w.Description != "" {
		detailWidth := m.width - 4
		if detailWidth < 40 {
			detailWidth = 60
		}
		wrapped := lipgloss.NewStyle().Width(detailWidth).Render(w.Description)
		for _, line := range strings.Split(wrapped, "\n") {
			b.WriteString(" " + normalStyle.Render(line) + "\n")
		}
	}

	b.WriteString("\n " + metaStyle.Render(w.RepositoryURL) + "\n")

	if m.statusMsg != "" {
		b.WriteString("\n " + upvoteStyle.Render(m.statusMsg) + "\n")
	}

	return truncateToHeight(b.String(), m.height)
}

// formatCommentTime formats a comment timestamp as a short relative or absolute string.
func formatCommentTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func formatNum(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
