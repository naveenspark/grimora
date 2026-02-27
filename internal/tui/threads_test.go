package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/naveenspark/grimora/pkg/domain"
)

func newTestThreadsModel() threadsModel {
	m := newThreadsModel(nil)
	m.width = 80
	m.height = 24
	m.myLogin = "testuser"
	return m
}

func makeTestThread(login, guild, lastMsg string) domain.Thread {
	return domain.Thread{
		ID:           uuid.New(),
		OtherLogin:   login,
		OtherGuildID: guild,
		LastMessage:  lastMsg,
		CreatedAt:    time.Now(),
	}
}

func TestThreadsListRendersRows(t *testing.T) {
	m := newTestThreadsModel()
	threads := []domain.Thread{
		makeTestThread("alice", "loomari", "Hey there!"),
		makeTestThread("bob", "cipher", "Check this out"),
	}
	m, _ = m.Update(threadsListLoadedMsg{threads: threads})

	view := m.View()
	if !strings.Contains(view, "alice") {
		t.Errorf("expected 'alice' in threads view, got:\n%s", view)
	}
	if !strings.Contains(view, "bob") {
		t.Errorf("expected 'bob' in threads view, got:\n%s", view)
	}
	if !strings.Contains(view, "Hey there!") {
		t.Errorf("expected last message preview in view, got:\n%s", view)
	}
}

func TestThreadsEmptyState(t *testing.T) {
	m := newTestThreadsModel()
	m, _ = m.Update(threadsListLoadedMsg{threads: nil})

	view := m.View()
	if !strings.Contains(view, "no threads yet") {
		t.Errorf("expected 'no threads yet' in empty threads view, got:\n%s", view)
	}
}

func TestThreadsEnterOpensConvo(t *testing.T) {
	m := newTestThreadsModel()
	threads := []domain.Thread{makeTestThread("alice", "loomari", "Hi")}
	m, _ = m.Update(threadsListLoadedMsg{threads: threads})

	// Press enter to open thread
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != threadsConvoState {
		t.Errorf("expected threadsConvoState after enter, got %d", m.state)
	}
	if m.openThreadLogin != "alice" {
		t.Errorf("expected openThreadLogin='alice', got %q", m.openThreadLogin)
	}
	if cmd == nil {
		t.Error("expected load messages command, got nil")
	}
}

func TestThreadsEscReturnsToList(t *testing.T) {
	m := newTestThreadsModel()
	threads := []domain.Thread{makeTestThread("alice", "loomari", "Hi")}
	m, _ = m.Update(threadsListLoadedMsg{threads: threads})

	// Open convo
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != threadsConvoState {
		t.Fatal("expected threadsConvoState")
	}

	// Unfocus input first
	m.inputFocused = false

	// Press esc to return to list
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != threadsListState {
		t.Errorf("expected threadsListState after esc, got %d", m.state)
	}
}

func TestThreadsSendReturnsCmd(t *testing.T) {
	m := newTestThreadsModel()
	m.state = threadsConvoState
	m.openThreadID = uuid.New().String()
	m.inputFocused = true
	m.input = "Hello!"

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected send command on enter with non-empty input, got nil")
	}
	if m.input != "" {
		t.Errorf("expected input cleared after send, got %q", m.input)
	}
}

func TestThreadsConvoRendersMessages(t *testing.T) {
	m := newTestThreadsModel()
	m.state = threadsConvoState
	m.openThreadID = uuid.New().String()
	m.openThreadLogin = "alice"
	m.openThreadGuild = "loomari"
	m.messages = []domain.Message{
		{
			ID:          uuid.New(),
			SenderLogin: "alice",
			Body:        "Hello from alice!",
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			SenderLogin: "testuser",
			Body:        "Hey alice!",
			CreatedAt:   time.Now(),
		},
	}

	view := m.View()
	if !strings.Contains(view, "Hello from alice!") {
		t.Errorf("expected alice's message in convo view, got:\n%s", view)
	}
	if !strings.Contains(view, "Hey alice!") {
		t.Errorf("expected own message in convo view, got:\n%s", view)
	}
}
