package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

// mentionRe matches @word patterns in message text.
var mentionRe = regexp.MustCompile(`@(\w+)`)

// hallPollInterval is how often the Hall polls for new messages.
const hallPollInterval = 3 * time.Second

// hallAnimInterval is the frame rate for card entrance animations.
const hallAnimInterval = 80 * time.Millisecond

// hallAnimFrames is the total number of animation frames (80ms * 20 = 1.6s).
const hallAnimFrames = 20

// hallTickMsg fires on each poll interval.
type hallTickMsg time.Time

// hallAnimTickMsg fires on each animation frame interval.
type hallAnimTickMsg time.Time

// cursorBlinkMsg toggles the input cursor on/off.
type cursorBlinkMsg struct{}

func cursorBlinkCmd() tea.Cmd {
	return tea.Tick(530*time.Millisecond, func(time.Time) tea.Msg {
		return cursorBlinkMsg{}
	})
}

func hallTickCmd() tea.Cmd {
	return tea.Tick(hallPollInterval, func(t time.Time) tea.Msg {
		return hallTickMsg(t)
	})
}

func hallAnimTickCmd() tea.Cmd {
	return tea.Tick(hallAnimInterval, func(t time.Time) tea.Msg {
		return hallAnimTickMsg(t)
	})
}

// hallMessagesMsg carries a batch of room messages from the API.
type hallMessagesMsg struct {
	messages []domain.RoomMessage
	err      error
}

// hallPresenceMsg carries room presence data from the API.
type hallPresenceMsg struct {
	count  int
	logins []string
	err    error
}

// hallSendMsg carries the result of a send attempt.
type hallSendMsg struct {
	err error
}

// hallProjectsMsg carries the user's workshop projects for # autocomplete.
type hallProjectsMsg struct {
	projects []domain.WorkshopProject
	err      error
}

// reactionCount is an emoji + count for display.
type reactionCount struct {
	Emoji string
	Count int
}

// chatMessage is a rendered message ready for display.
type chatMessage struct {
	ID          string
	SenderLogin string
	SenderGuild string
	Body        string
	Kind        string
	Metadata    map[string]string
	CreatedAt   time.Time
	IsSystem    bool
	IsGrimoire  bool
	IsSelf      bool
	Reactions   []reactionCount
	// Animation state (Phase 5)
	animFrame int
	animStart time.Time
}

// hallReactionsMsg carries batch reaction counts from the API.
type hallReactionsMsg struct {
	reactions map[string][]reactionCount
	err       error
}

// hallModel is the Hall (group chat) tab model.
// It polls /api/rooms/hall/messages every 3 seconds and renders messages
// in a scrollable log with an inline text input at the bottom.
type hallModel struct {
	client         *client.Client
	messages       []chatMessage
	input          string
	status         string // ephemeral status line (e.g. "sending not yet implemented")
	err            string
	connected      bool
	inputFocused   bool
	width          int
	height         int
	scroll         int    // lines scrolled up from bottom (0 = at bottom)
	myLogin        string // populated from the App.me after first load
	seenIDs        map[string]bool
	presenceCount  int
	presenceLogins []string
	cursorOn       bool // blink state for input cursor

	// @mention autocomplete state
	mentionActive  bool
	mentionQuery   string
	mentionMatches []string
	mentionCursor  int

	// #project autocomplete state
	projectActive  bool
	projectQuery   string
	projectMatches []domain.WorkshopProject
	projectCursor  int
	myProjects     []domain.WorkshopProject
}

func newHallModel(c *client.Client) hallModel {
	return hallModel{
		client:       c,
		seenIDs:      make(map[string]bool),
		inputFocused: true,
	}
}

func (m hallModel) Init() tea.Cmd {
	return tea.Batch(m.loadMessages(), m.loadProjects(), cursorBlinkCmd(), hallAnimTickCmd())
}

// loadProjects fetches the user's workshop projects for # autocomplete.
func (m hallModel) loadProjects() tea.Cmd {
	c := m.client
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		projects, err := c.ListWorkshopProjects(context.Background())
		return hallProjectsMsg{projects: projects, err: err}
	}
}

// hallSlug is the default public chat room.
const hallSlug = "the-hall"

// loadMessages fetches the 50 most recent messages and presence from the hall room.
func (m hallModel) loadMessages() tea.Cmd {
	c := m.client
	fetchMsgs := func() tea.Msg {
		msgs, err := c.GetRoomMessages(context.Background(), hallSlug, time.Time{}, 50)
		return hallMessagesMsg{messages: msgs, err: err}
	}
	fetchPresence := func() tea.Msg {
		p, err := c.GetRoomPresence(context.Background(), hallSlug)
		if err != nil {
			return hallPresenceMsg{err: err}
		}
		return hallPresenceMsg{count: p.Count, logins: p.Magicians}
	}
	return tea.Batch(fetchMsgs, fetchPresence)
}

// sendRoomMessage sends a message to the hall via REST POST.
func (m hallModel) sendRoomMessage(body string) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		_, err := c.SendRoomMessage(context.Background(), hallSlug, body)
		return hallSendMsg{err: err}
	}
}

// loadReactions fetches reaction counts for all currently loaded messages.
func (m hallModel) loadReactions() tea.Cmd {
	c := m.client
	ids := make([]string, 0, len(m.messages))
	for _, msg := range m.messages {
		ids = append(ids, msg.ID)
	}
	return func() tea.Msg {
		counts, err := c.GetReactionCounts(context.Background(), hallSlug, ids)
		if err != nil {
			return hallReactionsMsg{err: err}
		}
		result := make(map[string][]reactionCount, len(counts))
		for msgID, rcs := range counts {
			converted := make([]reactionCount, len(rcs))
			for i, rc := range rcs {
				converted[i] = reactionCount{Emoji: rc.Emoji, Count: rc.Count}
			}
			result[msgID] = converted
		}
		return hallReactionsMsg{reactions: result}
	}
}

func (m hallModel) Update(msg tea.Msg) (hallModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// Triggered when the App sets the current user's login after GetMe resolves.
	case meLoadedMsg:
		if msg.err == nil && msg.me != nil {
			m.myLogin = msg.me.GitHubLogin
			// Re-classify any already-loaded messages as self.
			for i := range m.messages {
				m.messages[i].IsSelf = (m.messages[i].SenderLogin == m.myLogin)
			}
		}

	case hallMessagesMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			// Keep polling even on error — transient network issues are common.
			return m, hallTickCmd()
		}
		m.err = ""
		m.connected = true

		// Merge new messages (de-duplicate by ID, then sort by time).
		for _, raw := range msg.messages {
			id := raw.ID.String()
			if m.seenIDs[id] {
				continue
			}
			m.seenIDs[id] = true

			// Parse metadata JSON
			var meta map[string]string
			if len(raw.Metadata) > 0 {
				_ = json.Unmarshal(raw.Metadata, &meta) //nolint:errcheck // best-effort parse
			}

			kind := raw.Kind
			if kind == "" {
				kind = "message"
			}

			cm := chatMessage{
				ID:          id,
				SenderLogin: raw.SenderLogin,
				SenderGuild: raw.SenderGuild,
				Body:        raw.Body,
				Kind:        kind,
				Metadata:    meta,
				CreatedAt:   raw.CreatedAt,
				IsSelf:      (raw.SenderLogin == m.myLogin),
			}

			// Animate new rich messages
			if kind != "message" && kind != "" {
				cm.animFrame = 1
				cm.animStart = time.Now()
			}

			m.messages = append(m.messages, cm)
		}

		// Sort chronologically — oldest first, newest at bottom near input.
		sort.Slice(m.messages, func(i, j int) bool {
			return m.messages[i].CreatedAt.Before(m.messages[j].CreatedAt)
		})

		// Keep the slice bounded to the most recent 200 messages.
		if len(m.messages) > 200 {
			trimmed := make([]chatMessage, 200)
			copy(trimmed, m.messages[len(m.messages)-200:])
			m.messages = trimmed
			// Rebuild seenIDs to match trimmed messages.
			m.seenIDs = make(map[string]bool, len(m.messages))
			for _, msg := range m.messages {
				m.seenIDs[msg.ID] = true
			}
		}

		// Fetch reaction counts for loaded messages.
		cmds := []tea.Cmd{hallTickCmd()}
		if m.client != nil && len(m.messages) > 0 {
			cmds = append(cmds, m.loadReactions())
		}
		return m, tea.Batch(cmds...)

	case hallReactionsMsg:
		if msg.err == nil && msg.reactions != nil {
			for i := range m.messages {
				if rcs, ok := msg.reactions[m.messages[i].ID]; ok {
					m.messages[i].Reactions = rcs
				}
			}
		}
		return m, nil

	case hallPresenceMsg:
		if msg.err == nil {
			m.presenceCount = msg.count
			m.presenceLogins = msg.logins
		}
		return m, nil

	case hallProjectsMsg:
		if msg.err == nil {
			m.myProjects = msg.projects
		}
		return m, nil

	case hallSendMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			return m, nil
		}
		m.status = ""
		return m, m.loadMessages()

	case hallTickMsg:
		return m, m.loadMessages()

	case cursorBlinkMsg:
		if m.inputFocused {
			m.cursorOn = !m.cursorOn
		}
		return m, cursorBlinkCmd()

	case hallAnimTickMsg:
		active := false
		for i := range m.messages {
			f := m.messages[i].animFrame
			if f > 0 && f <= hallAnimFrames {
				m.messages[i].animFrame++
				active = true
				if m.messages[i].animFrame > hallAnimFrames {
					m.messages[i].animFrame = 0 // static
				}
			}
		}
		if active {
			return m, hallAnimTickCmd()
		}
		return m, hallAnimTickCmd() // keep ticking to catch new messages

	case tea.KeyMsg:
		// Any keypress resets cursor to visible
		m.cursorOn = true
		if m.inputFocused {
			return m.updateInput(msg)
		}
		return m.updateNav(msg)
	}

	return m, nil
}

// updateInput handles key events when the text input is focused.
func (m hallModel) updateInput(msg tea.KeyMsg) (hallModel, tea.Cmd) {
	key := msg.String()

	// --- Mention autocomplete active ---
	if m.mentionActive {
		switch key {
		case "tab":
			// Accept selected suggestion
			if len(m.mentionMatches) > 0 {
				selected := m.mentionMatches[m.mentionCursor]
				// Replace trailing @query with @fullname + space
				m.input = strings.TrimSuffix(m.input, "@"+m.mentionQuery) + "@" + selected + " "
			}
			m.mentionActive = false
			m.mentionQuery = ""
			m.mentionMatches = nil
			m.mentionCursor = 0
			return m, nil
		case "up":
			if m.mentionCursor > 0 {
				m.mentionCursor--
			}
			return m, nil
		case "down":
			if m.mentionCursor < len(m.mentionMatches)-1 {
				m.mentionCursor++
			}
			return m, nil
		case "esc", " ":
			if key == " " {
				m.input += " "
			}
			m.mentionActive = false
			m.mentionQuery = ""
			m.mentionMatches = nil
			m.mentionCursor = 0
			return m, nil
		case "backspace":
			if m.mentionQuery == "" {
				// Remove the @ itself
				m.input = strings.TrimSuffix(m.input, "@")
				m.mentionActive = false
				m.mentionQuery = ""
				m.mentionMatches = nil
				m.mentionCursor = 0
			} else {
				// Shorten query
				m.mentionQuery = editRune(m.mentionQuery, "backspace")
				m.input = editRune(m.input, "backspace")
				m.mentionMatches = m.filterLogins(m.mentionQuery)
				m.mentionCursor = 0
			}
			return m, nil
		case "enter":
			// Accept if there are matches, otherwise send message
			if len(m.mentionMatches) > 0 {
				selected := m.mentionMatches[m.mentionCursor]
				m.input = strings.TrimSuffix(m.input, "@"+m.mentionQuery) + "@" + selected + " "
				m.mentionActive = false
				m.mentionQuery = ""
				m.mentionMatches = nil
				m.mentionCursor = 0
				return m, nil
			}
			m.mentionActive = false
			// Fall through to normal enter handling below
		default:
			// Append character to query
			if len(key) == 1 {
				m.mentionQuery += key
				m.input += key
				m.mentionMatches = m.filterLogins(m.mentionQuery)
				m.mentionCursor = 0
				if len(m.mentionMatches) == 0 {
					m.mentionActive = false
					m.mentionQuery = ""
					m.mentionMatches = nil
					m.mentionCursor = 0
				}
				return m, nil
			}
			// Non-printable: ignore for autocomplete
			return m, nil
		}
	}

	// --- Project autocomplete active ---
	if m.projectActive {
		switch key {
		case "tab", "enter":
			if len(m.projectMatches) > 0 {
				selected := m.projectMatches[m.projectCursor]
				tag := strings.Fields(selected.Name)[0]
				m.input = strings.TrimSuffix(m.input, "#"+m.projectQuery) + "#" + tag + " "
			}
			m.projectActive = false
			m.projectQuery = ""
			m.projectMatches = nil
			m.projectCursor = 0
			return m, nil
		case "up":
			if m.projectCursor > 0 {
				m.projectCursor--
			}
			return m, nil
		case "down":
			if m.projectCursor < len(m.projectMatches)-1 {
				m.projectCursor++
			}
			return m, nil
		case "esc", " ":
			if key == " " {
				m.input += " "
			}
			m.projectActive = false
			m.projectQuery = ""
			m.projectMatches = nil
			m.projectCursor = 0
			return m, nil
		case "backspace":
			if m.projectQuery == "" {
				m.input = strings.TrimSuffix(m.input, "#")
				m.projectActive = false
				m.projectQuery = ""
				m.projectMatches = nil
				m.projectCursor = 0
			} else {
				m.projectQuery = editRune(m.projectQuery, "backspace")
				m.input = editRune(m.input, "backspace")
				m.projectMatches = m.filterProjects(m.projectQuery)
				m.projectCursor = 0
				if len(m.projectMatches) == 0 {
					m.projectActive = false
					m.projectQuery = ""
					m.projectMatches = nil
					m.projectCursor = 0
				}
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.projectQuery += key
				m.input += key
				m.projectMatches = m.filterProjects(m.projectQuery)
				m.projectCursor = 0
				if len(m.projectMatches) == 0 {
					m.projectActive = false
					m.projectQuery = ""
					m.projectMatches = nil
					m.projectCursor = 0
				}
				return m, nil
			}
			return m, nil
		}
	}

	// --- Normal input handling ---
	switch key {
	case "esc":
		m.inputFocused = false
		m.status = ""
		return m, nil

	case "enter":
		body := strings.TrimSpace(m.input)
		if body == "" {
			return m, nil
		}
		if m.myLogin == "" {
			m.status = "run: grimora login"
			return m, nil
		}
		m.input = ""
		m.status = ""
		cmds := []tea.Cmd{m.sendRoomMessage(body)}
		// Refresh projects after /build so # picks it up
		if strings.HasPrefix(body, "/build ") {
			cmds = append(cmds, m.loadProjects())
		}
		return m, tea.Batch(cmds...)

	case "@":
		m.input += "@"
		m.mentionActive = true
		m.mentionQuery = ""
		m.mentionMatches = m.filterLogins("")
		m.mentionCursor = 0
		if len(m.mentionMatches) == 0 {
			m.mentionActive = false
		}
		return m, nil

	case "#":
		m.input += "#"
		if len(m.myProjects) > 0 {
			m.projectActive = true
			m.projectQuery = ""
			m.projectMatches = m.filterProjects("")
			m.projectCursor = 0
		}
		return m, nil

	default:
		m.input = editRune(m.input, key)
		return m, nil
	}
}

// updateNav handles key events when the input is not focused (scroll mode).
func (m hallModel) updateNav(msg tea.KeyMsg) (hallModel, tea.Cmd) {
	switch msg.String() {
	case "j":
		// Scroll down (toward bottom).
		if m.scroll > 0 {
			m.scroll--
		}
	case "k":
		// Scroll up (toward top), with ceiling.
		maxScroll := len(m.messages) * 3
		if m.scroll < maxScroll {
			m.scroll++
		}
	case "enter", "i", "/":
		m.inputFocused = true
		m.cursorOn = true
		m.status = ""
	}
	return m, nil
}

// View renders the Hall tab.
func (m hallModel) View() string {
	var b strings.Builder

	// Reserve lines: input(1) + presence(1 if logged in) + status(0-1) + autocomplete.
	chrome := 1 // input
	if m.myLogin != "" {
		chrome++ // presence line
	}
	if m.status != "" {
		chrome++
	}
	// Slash hints and autocomplete popups steal lines from the message viewport.
	if strings.HasPrefix(m.input, "/") && m.inputFocused {
		chrome += m.countSlashHints()
	}
	mentionLines := 0
	if m.mentionActive && len(m.mentionMatches) > 0 {
		mentionLines = len(m.mentionMatches)
		if mentionLines > 5 {
			mentionLines = 5
		}
		chrome += mentionLines
	}
	if m.projectActive && len(m.projectMatches) > 0 {
		projectLines := len(m.projectMatches)
		if projectLines > 5 {
			projectLines = 5
		}
		chrome += projectLines
	}
	viewportHeight := m.height - chrome
	if viewportHeight < 2 {
		viewportHeight = 2
	}

	// --- Message area ---
	if m.err != "" && len(m.messages) == 0 {
		padLines(viewportHeight-1, &b)
		b.WriteString(" " + dimStyle.Render("could not connect · check your connection or run: grimora login") + "\n")
	} else if m.myLogin == "" && !m.connected {
		padLines(viewportHeight-1, &b)
		b.WriteString(" " + dimStyle.Render("connecting...") + "\n")
	} else if len(m.messages) == 0 && m.err == "" {
		padLines(viewportHeight-1, &b)
		b.WriteString(" " + dimStyle.Render("no messages yet") + "\n")
	} else {
		b.WriteString(m.renderMessages(viewportHeight))
	}

	// --- Slash command hint popup ---
	if strings.HasPrefix(m.input, "/") && m.inputFocused {
		b.WriteString(m.renderSlashHints())
	}

	// --- Mention autocomplete popup ---
	if m.mentionActive && len(m.mentionMatches) > 0 {
		b.WriteString(m.renderMentionPopup())
	}

	// --- Project autocomplete popup ---
	if m.projectActive && len(m.projectMatches) > 0 {
		b.WriteString(m.renderProjectPopup())
	}

	// --- Presence line (username only; online count is in the tab bar) ---
	if m.myLogin != "" {
		b.WriteString(" " + dimStyle.Render(m.myLogin) + "\n")
	}

	// --- Input line ---
	b.WriteString(m.renderInput())
	b.WriteByte('\n')

	// --- Status line (transient only; static hints live in the global help bar) ---
	if m.status != "" {
		b.WriteString(" " + dimStyle.Render(m.status))
	}

	return b.String()
}

// renderMessages renders the message log clipped to viewportHeight lines,
// respecting the scroll offset. Newest messages appear at the bottom.
func (m hallModel) renderMessages(viewportHeight int) string {
	if len(m.messages) == 0 {
		return ""
	}

	// Build all rendered lines from all messages (wrapped messages produce multiple lines).
	var allLines []string
	for _, msg := range m.messages {
		rendered := m.renderMessage(msg)
		allLines = append(allLines, strings.Split(rendered, "\n")...)
		if len(msg.Reactions) > 0 {
			allLines = append(allLines, renderReactionLine(msg.Reactions))
		}
	}

	total := len(allLines)

	// Clamp scroll so we can't scroll past the top.
	maxScroll := total - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.scroll
	if scroll > maxScroll {
		scroll = maxScroll
	}

	// The window ends at (total - scroll), starts viewportHeight before that.
	end := total - scroll
	start := end - viewportHeight
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}

	visible := allLines[start:end]

	var b strings.Builder
	// Pad top with blank lines if there aren't enough messages to fill the viewport.
	for i := len(visible); i < viewportHeight; i++ {
		b.WriteByte('\n')
	}
	for _, line := range visible {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// renderMessage renders a single chat message, wrapping body text to fit the terminal width.
// May return multiple newline-separated lines for wrapped messages.
func (m hallModel) renderMessage(msg chatMessage) string {
	// System messages: centered "— text —"
	if msg.IsSystem {
		centered := fmt.Sprintf("— %s —", msg.Body)
		return " " + chatSysStyle.Render(centered)
	}

	// Rich message rendering by kind
	switch msg.Kind {
	case "build-start":
		return m.renderBuildStart(msg)
	case "build-update":
		return m.renderBuildUpdate(msg)
	case "ship":
		return m.renderShipCard(msg)
	case "seek":
		return m.renderSeek(msg)
	case "forge-verdict":
		return m.renderForgeVerdict(msg)
	case "cast":
		return m.renderCast(msg)
	}

	// Default: plain message
	return m.renderPlainMessage(msg)
}

// metaTitle returns the "title" from Metadata, falling back to Body.
func (msg chatMessage) metaTitle() string {
	if t, ok := msg.Metadata["title"]; ok && t != "" {
		return t
	}
	return msg.Body
}

// renderPlainMessage renders a normal chat message.
func (m hallModel) renderPlainMessage(msg chatMessage) string {
	timeStr := fmt.Sprintf("%8s", formatChatTime(msg.CreatedAt))
	timePart := metaStyle.Render(timeStr)
	sep := chatSepStyle.Render(" · ")

	var namePart string
	if msg.IsSelf {
		namePart = chatSelfNameStyle.Render("you")
	} else {
		name := msg.SenderLogin
		if msg.SenderGuild != "" {
			namePart = GuildStyle(msg.SenderGuild).Render(name)
		} else {
			namePart = chatTextStyle.Render(name)
		}
	}

	renderBody := func(s string) string {
		highlighted := renderBodyWithMentions(s, m.myLogin, msg.IsSelf)
		if msg.IsSelf {
			return chatSelfTextStyle.Render(highlighted)
		}
		return chatTextStyle.Render(highlighted)
	}

	bodyWidth := m.width - 26
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	wrapped := lipgloss.NewStyle().Width(bodyWidth).Render(msg.Body)
	lines := strings.Split(wrapped, "\n")

	result := " " + timePart + "  " + namePart + sep + renderBody(lines[0])
	if len(lines) > 1 {
		indent := strings.Repeat(" ", 15)
		for _, line := range lines[1:] {
			result += "\n" + indent + renderBody(line)
		}
	}
	return result
}

// renderBuildStart renders a compact build-start line.
func (m hallModel) renderBuildStart(msg chatMessage) string {
	title := msg.metaTitle()
	return " " + forgeStyle.Render("🔨") + " " + forgeStyle.Render(msg.SenderLogin) + " started a build · " + dimStyle.Render(truncStr(cleanTitle(title), 50))
}

// renderShipCard renders a multi-line ship card with animation (the only box card).
func (m hallModel) renderShipCard(msg chatMessage) string {
	const color = "#d4a844" // gold
	title := msg.metaTitle()
	label := goldStyle.Render("✦ SHIPPED")
	top := cardBorder("top", label, color, msg.animFrame, m.width)
	bar := goldStyle.Render(" │")
	body := bar + "  " + goldStyle.Render(msg.SenderLogin+" shipped ") + goldStyle.Bold(true).Render(`"`+truncStr(cleanTitle(title), 50)+`"`)
	bottom := cardBorder("bottom", "", color, msg.animFrame, m.width)
	return top + "\n" + body + "\n" + bottom
}

// renderSeek renders a compact seek line.
func (m hallModel) renderSeek(msg chatMessage) string {
	return " " + dimStyle.Render("✧") + " " + goldStyle.Render(msg.SenderLogin) + " seeks · " + chatTextStyle.Render(truncStr(msg.Body, 60))
}

// renderBuildUpdate renders a compact build update (no box).
func (m hallModel) renderBuildUpdate(msg chatMessage) string {
	return "   " + accentStyle.Render("⚡") + " " + accentStyle.Render(msg.SenderLogin) + " · " + dimStyle.Render(msg.Body)
}

// renderForgeVerdict renders a compact forge verdict line.
func (m hallModel) renderForgeVerdict(msg chatMessage) string {
	title := msg.metaTitle()
	potency := msg.Metadata["potency"]
	line := " " + accentStyle.Render("⚡") + " " + accentStyle.Render(msg.SenderLogin) + " forged · " + goldStyle.Render(`"`+truncStr(cleanTitle(title), 50)+`"`)
	if potency != "" {
		line += " " + potencyStyle(potencyFromStr(potency)).Render("P"+potency)
	}
	return line
}

// renderCast renders a compact Grimoire cast line.
func (m hallModel) renderCast(msg chatMessage) string {
	label := grimLabelStyle.Render("Grimoire:")
	bodyWidth := m.width - 16
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	wrapped := lipgloss.NewStyle().Width(bodyWidth).Render(msg.Body)
	lines := strings.Split(wrapped, "\n")
	result := " " + castStyle.Render("✦") + " " + label + " " + grimVoiceStyle.Render(lines[0])
	if len(lines) > 1 {
		indent := strings.Repeat(" ", 15)
		for _, line := range lines[1:] {
			result += "\n" + indent + grimVoiceStyle.Render(line)
		}
	}
	return result
}

// potencyFromStr converts a string potency to int, defaulting to 1.
func potencyFromStr(s string) int {
	switch s {
	case "3":
		return 3
	case "2":
		return 2
	default:
		return 1
	}
}

// renderInput renders the text input line at the bottom.
func (m hallModel) renderInput() string {
	prompt := inputPromptStyle.Render("> ")
	placeholder := "say something..."
	if m.myLogin == "" {
		placeholder = "grimora login to chat"
	}
	if !m.inputFocused {
		if m.input == "" {
			return " " + prompt + inputPlaceholderStyle.Render(placeholder)
		}
		return " " + prompt + dimStyle.Render(m.input)
	}
	// Focused: show text with blinking cursor block.
	text := m.input
	cursor := " " // invisible phase
	if m.cursorOn {
		cursor = accentStyle.Render("█")
	}
	if text == "" {
		return " " + prompt + cursor
	}
	return " " + prompt + chatSelfTextStyle.Render(text) + cursor
}

// slashCommands defines the available slash commands and their descriptions.
var slashCommands = []struct {
	cmd  string
	desc string
}{
	{"/build <title>", "start a build"},
	{"/b <update>", "update a build"},
	{"/ship <title>", "ship something"},
	{"/seek <question>", "ask for help"},
}

// countSlashHints returns the number of slash hint lines that will be rendered.
func (m hallModel) countSlashHints() int {
	prefix := strings.TrimPrefix(m.input, "/")
	n := 0
	for _, sc := range slashCommands {
		trimmedCmd := strings.TrimPrefix(sc.cmd, "/")
		if prefix != "" && !strings.HasPrefix(trimmedCmd, prefix) {
			continue
		}
		n++
	}
	return n
}

// renderSlashHints renders slash command hints above the input when typing "/".
func (m hallModel) renderSlashHints() string {
	prefix := strings.TrimPrefix(m.input, "/")
	var b strings.Builder
	for _, sc := range slashCommands {
		// Filter by prefix
		trimmedCmd := strings.TrimPrefix(sc.cmd, "/")
		if prefix != "" && !strings.HasPrefix(trimmedCmd, prefix) {
			continue
		}
		b.WriteString("   " + accentStyle.Render(sc.cmd) + "  " + dimStyle.Render(sc.desc) + "\n")
	}
	return b.String()
}

// renderMentionPopup renders the autocomplete suggestion list above the input line.
func (m hallModel) renderMentionPopup() string {
	var b strings.Builder
	limit := len(m.mentionMatches)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		login := m.mentionMatches[i]
		if i == m.mentionCursor {
			b.WriteString("   " + accentStyle.Render("▸ "+login))
		} else {
			b.WriteString("     " + dimStyle.Render(login))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// filterProjects returns projects matching the query prefix (case-insensitive).
func (m hallModel) filterProjects(query string) []domain.WorkshopProject {
	q := strings.ToLower(query)
	var matches []domain.WorkshopProject
	for _, p := range m.myProjects {
		if q == "" || strings.Contains(strings.ToLower(p.Name), q) {
			matches = append(matches, p)
		}
	}
	return matches
}

// renderProjectPopup renders the project autocomplete suggestion list.
func (m hallModel) renderProjectPopup() string {
	var b strings.Builder
	limit := len(m.projectMatches)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		name := m.projectMatches[i].Name
		if i == m.projectCursor {
			b.WriteString("   " + goldStyle.Render("▸ #"+name))
		} else {
			b.WriteString("     " + dimStyle.Render("#"+name))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// renderReactionLine renders a dim line of emoji counts indented to body start.
func renderReactionLine(reactions []reactionCount) string {
	var parts []string
	for _, r := range reactions {
		parts = append(parts, fmt.Sprintf("%s%d", r.Emoji, r.Count))
	}
	return "               " + dimStyle.Render(strings.Join(parts, " "))
}

// padLines writes blank lines to fill dead space above sparse message lists.
func padLines(n int, b *strings.Builder) {
	for i := 0; i < n; i++ {
		b.WriteByte('\n')
	}
}

// formatChatTime formats a message timestamp as a short wall-clock time (H:MM).
// For messages older than today it shows "NdAgo" to save column space.
func formatChatTime(t time.Time) string {
	now := time.Now()
	// Same calendar day.
	y1, mo1, d1 := t.Date()
	y2, mo2, d2 := now.Date()
	if y1 == y2 && mo1 == mo2 && d1 == d2 {
		return fmt.Sprintf("%d:%02d", t.Hour(), t.Minute())
	}
	days := int(now.Sub(t).Hours() / 24)
	if days < 1 {
		days = 1
	}
	return fmt.Sprintf("%dd ago", days)
}

// knownLogins returns a deduplicated, sorted list of logins from
// both presenceLogins and message senders.
func (m hallModel) knownLogins() []string {
	seen := make(map[string]bool)
	for _, l := range m.presenceLogins {
		if l != "" && l != m.myLogin {
			seen[l] = true
		}
	}
	for _, msg := range m.messages {
		if msg.SenderLogin != "" && msg.SenderLogin != m.myLogin && !msg.IsSystem {
			seen[msg.SenderLogin] = true
		}
	}
	logins := make([]string, 0, len(seen))
	for l := range seen {
		logins = append(logins, l)
	}
	sort.Strings(logins)
	return logins
}

// filterLogins returns logins matching the query prefix (case-insensitive).
// An empty query returns all known logins.
func (m hallModel) filterLogins(query string) []string {
	all := m.knownLogins()
	if query == "" {
		return all
	}
	q := strings.ToLower(query)
	var matches []string
	for _, l := range all {
		if strings.HasPrefix(strings.ToLower(l), q) {
			matches = append(matches, l)
		}
	}
	return matches
}

// renderBodyWithMentions highlights @mentions in message body text.
// Self-mentions get extra bright styling.
func renderBodyWithMentions(body, myLogin string, isSelf bool) string {
	return mentionRe.ReplaceAllStringFunc(body, func(match string) string {
		login := match[1:] // strip leading @
		if strings.EqualFold(login, myLogin) {
			return mentionSelfStyle.Render(match)
		}
		return mentionStyle.Render(match)
	})
}
