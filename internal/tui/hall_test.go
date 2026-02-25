package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/naveenspark/grimora/pkg/domain"
)

func newTestHallModel() hallModel {
	m := newHallModel(nil)
	m.width = 80
	m.height = 24
	return m
}

func makeTestRoomMessage(login, guild, body string) domain.RoomMessage {
	return domain.RoomMessage{
		ID:          uuid.New(),
		SenderLogin: login,
		SenderGuild: guild,
		Body:        body,
		CreatedAt:   time.Now(),
	}
}

func TestHallMessagesLoadedRendersMessageList(t *testing.T) {
	m := newTestHallModel()
	msgs := []domain.RoomMessage{
		makeTestRoomMessage("alice", "loomari", "Hello from the hall!"),
		makeTestRoomMessage("bob", "cipher", "Greetings, fellow mages."),
	}
	m, _ = m.Update(hallMessagesMsg{messages: msgs})

	view := m.View()
	if !strings.Contains(view, "alice") {
		t.Errorf("expected 'alice' in hall view, got:\n%s", view)
	}
	if !strings.Contains(view, "Hello from the hall!") {
		t.Errorf("expected message body in hall view, got:\n%s", view)
	}
}

func TestHallSelfMessageHighlightedDifferently(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "me"

	msgs := []domain.RoomMessage{
		makeTestRoomMessage("me", "loomari", "This is my message"),
		makeTestRoomMessage("other", "cipher", "This is their message"),
	}
	m, _ = m.Update(hallMessagesMsg{messages: msgs})

	view := m.View()
	// Self message should render "you" instead of login
	if !strings.Contains(view, "you") {
		t.Errorf("expected 'you' for self message in hall view, got:\n%s", view)
	}
}

func TestHallSystemMessageRenderedCentered(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "testuser"
	// Manually add a system message to the messages slice
	m.messages = append(m.messages, chatMessage{
		ID:       "sys-1",
		Body:     "User joined the hall",
		IsSystem: true,
	})

	view := m.View()
	if !strings.Contains(view, "User joined the hall") {
		t.Errorf("expected system message in hall view, got:\n%s", view)
	}
	// System messages are wrapped in dashes: — text —
	if !strings.Contains(view, "—") {
		t.Errorf("expected '—' around system message, got:\n%s", view)
	}
}

func TestHallEmptyStateShowsNoMessagesYet(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "testuser"
	m, _ = m.Update(hallMessagesMsg{messages: nil})

	view := m.View()
	if !strings.Contains(view, "no messages yet") {
		t.Errorf("expected 'no messages yet' in empty hall, got:\n%s", view)
	}
}

func TestHallConnectionLoadingState(t *testing.T) {
	m := newTestHallModel()
	// Fresh model, not connected yet
	view := m.View()
	if !strings.Contains(view, "connecting") {
		t.Errorf("expected 'connecting...' in hall view before load, got:\n%s", view)
	}
}

func TestHallConnectionErrorState(t *testing.T) {
	m := newTestHallModel()
	m, _ = m.Update(hallMessagesMsg{err: &testErr{msg: "network timeout"}})

	view := m.View()
	if !strings.Contains(view, "could not connect") {
		t.Errorf("expected 'could not connect' in hall view on connection error, got:\n%s", view)
	}
}

func TestHallConnectedState(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "testuser"
	m, _ = m.Update(hallMessagesMsg{messages: []domain.RoomMessage{}})

	if !m.connected {
		t.Error("expected connected=true after successful load, got false")
	}
}

func TestHallInputFocusToggle(t *testing.T) {
	m := newTestHallModel()
	// Default: inputFocused=true
	if !m.inputFocused {
		t.Fatal("expected inputFocused=true by default")
	}

	// Press Esc to unfocus
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.inputFocused {
		t.Error("expected inputFocused=false after Esc, got true")
	}

	// Press Enter to refocus
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.inputFocused {
		t.Error("expected inputFocused=true after Enter, got false")
	}
}

func TestHallSendTriggersCommand(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "sender"
	m.inputFocused = true
	m.input = "Hello hall"

	// Press Enter to send
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected send command on Enter with non-empty input, got nil")
	}
	// Input should be cleared
	if m.input != "" {
		t.Errorf("expected input cleared after send, got %q", m.input)
	}
}

func TestHallSendRequiresLogin(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "" // not logged in
	m.inputFocused = true
	m.input = "Hello"

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.status, "grimora login") {
		t.Errorf("expected 'grimora login' status when not logged in, got %q", m.status)
	}
}

func TestHallScrollNavigation(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "user"
	// Add some messages directly to bypass the deduplication path
	for i := 0; i < 10; i++ {
		id := uuid.New().String()
		m.seenIDs[id] = true
		m.messages = append(m.messages, chatMessage{
			ID:          id,
			SenderLogin: "other",
			Body:        "Message content",
		})
	}
	m.connected = true
	m.inputFocused = false // in nav mode

	// Scroll up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.scroll <= 0 {
		t.Errorf("expected scroll > 0 after 'k', got %d", m.scroll)
	}

	prevScroll := m.scroll
	// Scroll down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.scroll >= prevScroll {
		t.Errorf("expected scroll to decrease after 'j', got %d (was %d)", m.scroll, prevScroll)
	}
}

func TestHallInputTyping(t *testing.T) {
	m := newTestHallModel()
	m.inputFocused = true

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})

	if m.input != "hi" {
		t.Errorf("expected input='hi', got %q", m.input)
	}

	view := m.View()
	if !strings.Contains(view, "hi") {
		t.Errorf("expected typed text in hall view, got:\n%s", view)
	}
}

func TestHallDeduplicatesMessages(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "user"

	// Same message delivered twice
	rawMsg := makeTestRoomMessage("alice", "loomari", "Duplicate message")
	m, _ = m.Update(hallMessagesMsg{messages: []domain.RoomMessage{rawMsg}})
	initialCount := len(m.messages)

	// Deliver same message again (seenIDs prevents re-adding)
	m, _ = m.Update(hallMessagesMsg{messages: []domain.RoomMessage{rawMsg}})
	if len(m.messages) != initialCount {
		t.Errorf("expected deduplication: %d messages, got %d", initialCount, len(m.messages))
	}
}

func TestHallRenderMessageSelf(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "me"

	selfMsg := chatMessage{
		ID:          "self-1",
		SenderLogin: "me",
		Body:        "My message body",
		IsSelf:      true,
		CreatedAt:   time.Now(),
	}

	rendered := m.renderMessage(selfMsg)
	if !strings.Contains(rendered, "you") {
		t.Errorf("expected 'you' for self message render, got: %q", rendered)
	}
	if !strings.Contains(rendered, "My message body") {
		t.Errorf("expected body in self message render, got: %q", rendered)
	}
}

func TestHallTickTriggersReload(t *testing.T) {
	m := newTestHallModel()
	_, cmd := m.Update(hallTickMsg(time.Now()))
	if cmd == nil {
		t.Error("expected hallTickMsg to return a reload command, got nil")
	}
}

// testErr is a simple error type for tests.
type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
