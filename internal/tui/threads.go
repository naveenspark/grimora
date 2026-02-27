package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/naveenspark/grimora/pkg/client"
	"github.com/naveenspark/grimora/pkg/domain"
)

// threadsState distinguishes between list and conversation views.
type threadsState int

const (
	threadsListState  threadsState = iota
	threadsConvoState              // viewing a single thread
)

// threadsPollInterval is how often the open conversation polls for new messages.
const threadsPollInterval = 5 * time.Second

// -- messages --

type threadsListLoadedMsg struct {
	threads []domain.Thread
	err     error
}

type threadsMessagesLoadedMsg struct {
	threadID string
	messages []domain.Message
	err      error
}

type threadsSendMsg struct {
	err error
}

type threadsStartedMsg struct {
	thread *domain.Thread
	err    error
}

type threadsPollTickMsg time.Time

func threadsPollCmd() tea.Cmd {
	return tea.Tick(threadsPollInterval, func(t time.Time) tea.Msg {
		return threadsPollTickMsg(t)
	})
}

// -- model --

type threadsModel struct {
	client  *client.Client
	state   threadsState
	threads []domain.Thread
	cursor  int
	err     string
	width   int
	height  int
	myLogin string
	loading bool

	// convo state
	openThreadID    string
	openThreadLogin string
	openThreadGuild string
	messages        []domain.Message
	input           string
	inputFocused    bool
	cursorOn        bool
	status          string

	// new thread
	startInput string
}

func newThreadsModel(c *client.Client) threadsModel {
	return threadsModel{client: c}
}

func (m threadsModel) Init() tea.Cmd {
	return m.loadThreads()
}

func (m threadsModel) loadThreads() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		threads, err := c.ListThreads(context.Background())
		return threadsListLoadedMsg{threads: threads, err: err}
	}
}

func (m threadsModel) loadMessages() tea.Cmd {
	c := m.client
	threadID := m.openThreadID
	return func() tea.Msg {
		msgs, err := c.GetMessages(context.Background(), threadID, 50, 0)
		return threadsMessagesLoadedMsg{threadID: threadID, messages: msgs, err: err}
	}
}

func (m threadsModel) sendMessage(body string) tea.Cmd {
	c := m.client
	threadID := m.openThreadID
	return func() tea.Msg {
		_, err := c.SendMessage(context.Background(), threadID, body)
		return threadsSendMsg{err: err}
	}
}

func (m threadsModel) startThread(login string) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		thread, err := c.StartThread(context.Background(), login)
		return threadsStartedMsg{thread: thread, err: err}
	}
}

func (m threadsModel) Update(msg tea.Msg) (threadsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case meLoadedMsg:
		if msg.err == nil && msg.me != nil {
			m.myLogin = msg.me.GitHubLogin
		}

	case threadsListLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.threads = msg.threads
			m.err = ""
		}

	case threadsMessagesLoadedMsg:
		if msg.threadID == m.openThreadID {
			if msg.err != nil {
				m.status = "error loading messages"
			} else {
				m.messages = msg.messages
			}
		}
		if m.state == threadsConvoState {
			return m, threadsPollCmd()
		}

	case threadsSendMsg:
		if msg.err != nil {
			m.status = "send failed: " + msg.err.Error()
		} else {
			m.status = ""
			return m, m.loadMessages()
		}

	case threadsStartedMsg:
		if msg.err != nil {
			m.status = "failed: " + msg.err.Error()
		} else if msg.thread != nil {
			m.state = threadsConvoState
			m.openThreadID = msg.thread.ID.String()
			m.openThreadLogin = msg.thread.OtherLogin
			m.openThreadGuild = msg.thread.OtherGuildID
			m.inputFocused = true
			m.cursorOn = true
			m.input = ""
			m.startInput = ""
			return m, tea.Batch(m.loadMessages(), cursorBlinkCmd())
		}

	case threadsPollTickMsg:
		if m.state == threadsConvoState {
			return m, m.loadMessages()
		}

	case cursorBlinkMsg:
		if m.inputFocused {
			m.cursorOn = !m.cursorOn
		}
		return m, cursorBlinkCmd()

	case tea.KeyMsg:
		m.cursorOn = true
		switch m.state {
		case threadsListState:
			return m.updateList(msg)
		case threadsConvoState:
			return m.updateConvo(msg)
		}
	}
	return m, nil
}

func (m threadsModel) updateList(msg tea.KeyMsg) (threadsModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.threads)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(m.threads) > 0 && m.cursor < len(m.threads) {
			thread := m.threads[m.cursor]
			m.state = threadsConvoState
			m.openThreadID = thread.ID.String()
			m.openThreadLogin = thread.OtherLogin
			m.openThreadGuild = thread.OtherGuildID
			m.inputFocused = true
			m.cursorOn = true
			m.input = ""
			return m, tea.Batch(m.loadMessages(), cursorBlinkCmd())
		}
	case "s":
		// Start new thread — for now prompt isn't implemented, use peek logins
		// TODO: implement login input for new thread
		if len(m.threads) > 0 && m.cursor < len(m.threads) {
			login := m.threads[m.cursor].OtherLogin
			return m, m.startThread(login)
		}
	case "p":
		if len(m.threads) > 0 && m.cursor < len(m.threads) {
			login := m.threads[m.cursor].OtherLogin
			return m, func() tea.Msg { return showPeekMsg{login: login} }
		}
	case "r":
		return m, m.loadThreads()
	}
	return m, nil
}

func (m threadsModel) updateConvo(msg tea.KeyMsg) (threadsModel, tea.Cmd) {
	key := msg.String()

	if m.inputFocused {
		switch key {
		case "esc":
			m.inputFocused = false
			return m, nil
		case "enter":
			body := strings.TrimSpace(m.input)
			if body == "" {
				return m, nil
			}
			m.input = ""
			return m, m.sendMessage(body)
		default:
			m.input = editRune(m.input, key)
			return m, nil
		}
	}

	// Nav mode
	switch key {
	case "esc":
		m.state = threadsListState
		m.openThreadID = ""
		m.messages = nil
		m.input = ""
		return m, m.loadThreads()
	case "enter", "i":
		m.inputFocused = true
		m.cursorOn = true
		return m, nil
	}
	return m, nil
}

func (m threadsModel) View() string {
	switch m.state {
	case threadsConvoState:
		return m.viewConvo()
	default:
		return m.viewList()
	}
}

func (m threadsModel) viewList() string {
	var b strings.Builder

	b.WriteString(" " + presenceTitleStyle.Render("Threads") + "\n")

	sep := strings.Repeat("─", max(m.width-2, 4))
	b.WriteString(" " + metaStyle.Render(sep) + "\n")

	if m.loading {
		b.WriteString(" " + dimStyle.Render("loading...") + "\n")
		return b.String()
	}
	if m.err != "" {
		b.WriteString(" " + dimStyle.Render("error: "+m.err) + "\n")
		return b.String()
	}
	if len(m.threads) == 0 {
		b.WriteString("\n " + dimStyle.Render("no threads yet · press s to start one") + "\n")
		return b.String()
	}

	for i, thread := range m.threads {
		isActive := i == m.cursor
		cursor := "  "
		if isActive {
			cursor = accentStyle.Render("▸") + " "
		}

		loginStyled := GuildStyle(thread.OtherGuildID).Render(thread.OtherLogin)
		if isActive {
			loginStyled = selectedStyle.Render(thread.OtherLogin)
		}

		preview := truncStr(thread.LastMessage, 40)
		if preview == "" {
			preview = "no messages"
		}

		timeStr := formatTime(thread.CreatedAt)

		fmt.Fprintf(&b, " %s%s  %s  %s\n",
			cursor,
			loginStyled,
			dimStyle.Render(preview),
			metaStyle.Render(timeStr),
		)
	}

	if m.status != "" {
		b.WriteString("\n " + dimStyle.Render(m.status) + "\n")
	}

	return b.String()
}

func (m threadsModel) viewConvo() string {
	var b strings.Builder

	// Header
	loginStyled := GuildStyle(m.openThreadGuild).Render(m.openThreadLogin)
	b.WriteString(" " + presenceTitleStyle.Render("Thread with ") + loginStyled + "\n")

	sep := strings.Repeat("─", max(m.width-2, 4))
	b.WriteString(" " + metaStyle.Render(sep) + "\n")

	// Messages
	chrome := 4 // header + sep + input + status
	if m.status != "" {
		chrome++
	}
	viewportHeight := m.height - chrome
	if viewportHeight < 2 {
		viewportHeight = 2
	}

	if len(m.messages) == 0 {
		padLines(viewportHeight, &b)
		b.WriteString(" " + dimStyle.Render("no messages yet") + "\n")
	} else {
		var allLines []string
		for _, msg := range m.messages {
			line := m.renderThreadMessage(msg)
			allLines = append(allLines, strings.Split(line, "\n")...)
		}

		// Show last N lines
		total := len(allLines)
		start := total - viewportHeight
		if start < 0 {
			start = 0
		}
		visible := allLines[start:]

		// Pad top
		for i := len(visible); i < viewportHeight; i++ {
			b.WriteByte('\n')
		}
		for _, line := range visible {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	// Input
	b.WriteString(m.renderConvoInput())
	b.WriteByte('\n')

	if m.status != "" {
		b.WriteString(" " + dimStyle.Render(m.status))
	}

	return b.String()
}

func (m threadsModel) renderThreadMessage(msg domain.Message) string {
	timeStr := fmt.Sprintf("%8s", formatChatTime(msg.CreatedAt))
	timePart := metaStyle.Render(timeStr)
	sep := chatSepStyle.Render(" · ")

	var namePart string
	isSelf := msg.SenderLogin == m.myLogin
	if isSelf {
		namePart = chatSelfNameStyle.Render("you")
	} else {
		namePart = GuildStyle(m.openThreadGuild).Render(msg.SenderLogin)
	}

	bodyWidth := m.width - 26
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	wrapped := lipgloss.NewStyle().Width(bodyWidth).Render(msg.Body)
	lines := strings.Split(wrapped, "\n")

	bodyStyle := chatTextStyle
	if isSelf {
		bodyStyle = chatSelfTextStyle
	}

	result := " " + timePart + "  " + namePart + sep + bodyStyle.Render(lines[0])
	if len(lines) > 1 {
		indent := strings.Repeat(" ", 15)
		for _, line := range lines[1:] {
			result += "\n" + indent + bodyStyle.Render(line)
		}
	}
	return result
}

func (m threadsModel) renderConvoInput() string {
	const timeIndent = "          " // 10 spaces — matches timestamp + gap

	sep := chatSepStyle.Render(" · ")
	namePart := chatSelfNameStyle.Render("you")
	if !m.inputFocused {
		if m.input == "" {
			return timeIndent + namePart + sep + inputPlaceholderStyle.Render("type a message...")
		}
		return timeIndent + namePart + sep + dimStyle.Render(m.input)
	}
	cursor := " "
	if m.cursorOn {
		cursor = accentStyle.Render("█")
	}
	if m.input == "" {
		return timeIndent + namePart + sep + cursor
	}
	return timeIndent + namePart + sep + chatSelfTextStyle.Render(m.input) + cursor
}

func (m threadsModel) helpKeys() string {
	switch m.state {
	case threadsConvoState:
		if m.inputFocused {
			return helpEntry("enter", "send") + "  " + helpEntry("esc", "nav")
		}
		return helpEntry("enter", "type") + "  " + helpEntry("esc", "back")
	default:
		return helpEntry("j/k", "nav") + "  " + helpEntry("enter", "open") + "  " + helpEntry("s", "new") + "  " + helpEntry("p", "peek") + "  " + helpEntry("h", "help") + "  " + helpEntry("q", "quit")
	}
}
