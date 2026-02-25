package tui

import (
	"context"
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

// hallTickMsg fires on each poll interval.
type hallTickMsg time.Time

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

// chatMessage is a rendered message ready for display.
type chatMessage struct {
	ID          string
	SenderLogin string
	SenderGuild string
	Body        string
	CreatedAt   time.Time
	IsSystem    bool
	IsGrimoire  bool
	IsSelf      bool
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
}

func newHallModel(c *client.Client) hallModel {
	return hallModel{
		client:       c,
		seenIDs:      make(map[string]bool),
		inputFocused: true, // default: typing goes straight to input
		cursorOn:     true,
	}
}

func (m hallModel) Init() tea.Cmd {
	return tea.Batch(m.loadMessages(), cursorBlinkCmd())
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

		// Merge new messages (server returns newest-last; de-duplicate by ID).
		for _, raw := range msg.messages {
			id := raw.ID.String()
			if m.seenIDs[id] {
				continue
			}
			m.seenIDs[id] = true
			cm := chatMessage{
				ID:          id,
				SenderLogin: raw.SenderLogin,
				SenderGuild: raw.SenderGuild,
				Body:        raw.Body,
				CreatedAt:   raw.CreatedAt,
				IsSelf:      (raw.SenderLogin == m.myLogin),
			}
			m.messages = append(m.messages, cm)
		}

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

		return m, hallTickCmd()

	case hallPresenceMsg:
		if msg.err == nil {
			m.presenceCount = msg.count
			m.presenceLogins = msg.logins
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
		return m, m.sendRoomMessage(body)

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

	// Reserve lines: 1 header + (0 or 1 roster) + 1 separator + 1 topic + 1 input + (0 or 1 status) + autocomplete.
	hasRoster := m.presenceCount > 0 && len(m.presenceLogins) > 0
	chrome := 4
	if hasRoster {
		chrome = 5
	}
	if m.status != "" {
		chrome++
	}
	// Autocomplete popup steals lines from the message viewport.
	mentionLines := 0
	if m.mentionActive && len(m.mentionMatches) > 0 {
		mentionLines = len(m.mentionMatches)
		if mentionLines > 5 {
			mentionLines = 5
		}
		chrome += mentionLines
	}
	viewportHeight := m.height - chrome
	if viewportHeight < 2 {
		viewportHeight = 2
	}

	// --- Header ---
	b.WriteString(" ")
	b.WriteString(presenceTitleStyle.Render("The Hall"))
	if !m.connected && m.err == "" {
		b.WriteString("  " + dimStyle.Render("· connecting..."))
	} else if m.err != "" {
		b.WriteString("  " + dimStyle.Render("· could not connect"))
	} else if hasRoster {
		b.WriteString("  " + presenceDotStyle.Render("●") + " " + dimStyle.Render(fmt.Sprintf("%d here", m.presenceCount)))
	} else {
		b.WriteString("  " + dimStyle.Render("· connected"))
	}
	b.WriteByte('\n')

	// --- Presence roster ---
	if hasRoster {
		names := strings.Join(m.presenceLogins, dimStyle.Render(", "))
		b.WriteString("   " + dimStyle.Render(names) + "\n")
	}

	// --- Separator ---
	sep := strings.Repeat("─", max(m.width-2, 4))
	b.WriteString(" " + metaStyle.Render(sep) + "\n")

	// --- Topic line ---
	b.WriteString(" " + chatSysStyle.Render("the fire's always lit. talk shop, share what you're building, or just lurk.") + "\n")

	// --- Message area ---
	if m.myLogin == "" && !m.connected {
		// Not authed or not yet loaded.
		padLines(viewportHeight, &b)
		b.WriteString(" " + dimStyle.Render("run: grimora login") + "\n")
	} else if len(m.messages) == 0 && m.err == "" {
		padLines(viewportHeight, &b)
		b.WriteString(" " + dimStyle.Render("no messages yet") + "\n")
	} else if m.err != "" && len(m.messages) == 0 {
		padLines(viewportHeight, &b)
		b.WriteString(" " + dimStyle.Render("could not load messages · check your connection or run: grimora login") + "\n")
	} else {
		b.WriteString(m.renderMessages(viewportHeight))
	}

	// --- Mention autocomplete popup ---
	if m.mentionActive && len(m.mentionMatches) > 0 {
		b.WriteString(m.renderMentionPopup())
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

	// Time column: right-aligned in 8 chars.
	timeStr := fmt.Sprintf("%8s", formatChatTime(msg.CreatedAt))
	timePart := metaStyle.Render(timeStr)

	// Separator.
	sep := chatSepStyle.Render(" · ")

	// Name differs for self vs others.
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

	// Body styling helper — applies mention highlighting then base color
	renderBody := func(s string) string {
		highlighted := renderBodyWithMentions(s, m.myLogin, msg.IsSelf)
		if msg.IsSelf {
			return chatSelfTextStyle.Render(highlighted)
		}
		return chatTextStyle.Render(highlighted)
	}

	// Wrap body text to available width
	bodyWidth := m.width - 26
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	wrapped := lipgloss.NewStyle().Width(bodyWidth).Render(msg.Body)
	lines := strings.Split(wrapped, "\n")

	// First line: full prefix + body
	result := " " + timePart + "  " + namePart + sep + renderBody(lines[0])

	if len(lines) > 1 {
		// Continuation lines aligned under body start
		indent := strings.Repeat(" ", 15)
		for _, line := range lines[1:] {
			result += "\n" + indent + renderBody(line)
		}
	}

	return result
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
