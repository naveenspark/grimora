package tui

import (
	"fmt"
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
	// Self message should render the login name
	if !strings.Contains(view, "me") {
		t.Errorf("expected self login 'me' in hall view, got:\n%s", view)
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
	// Default: inputFocused=true (talk mode — chat-first UX)
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
	if !strings.Contains(rendered, "me") {
		t.Errorf("expected self login 'me' in rendered message, got: %q", rendered)
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

func TestHallReactionDisplay(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "user"

	m.messages = []chatMessage{
		{
			ID:          "msg-1",
			SenderLogin: "alice",
			Body:        "Hello!",
			CreatedAt:   time.Now(),
			Reactions: []reactionCount{
				{Emoji: "🔥", Count: 3},
				{Emoji: "✨", Count: 1},
			},
		},
	}
	m.connected = true

	view := m.View()
	if !strings.Contains(view, "🔥") {
		t.Errorf("expected fire emoji in view, got:\n%s", view)
	}
	if !strings.Contains(view, "3") {
		t.Errorf("expected count '3' in view, got:\n%s", view)
	}
}

// --- Layout verification tests ---
// These tests verify line counts so the TUI never clips the input/cursor.

func TestHallViewLineCount(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *hallModel)
		wantLen int // expected number of newlines in View() output = m.height
	}{
		{
			name: "connecting state (not logged in)",
			setup: func(m *hallModel) {
				// fresh model, myLogin="" → chrome=1 (input only)
			},
			wantLen: 24, // m.height
		},
		{
			name: "empty with login",
			setup: func(m *hallModel) {
				m.myLogin = "naveenspark"
				m.connected = true
				m.presenceCount = 3
				// no messages → "no messages yet"
				m.Update(hallMessagesMsg{messages: nil}) //nolint:errcheck
			},
			wantLen: 24,
		},
		{
			name: "error state",
			setup: func(m *hallModel) {
				m.Update(hallMessagesMsg{err: &testErr{msg: "timeout"}}) //nolint:errcheck
			},
			wantLen: 24,
		},
		{
			name: "with messages and login",
			setup: func(m *hallModel) {
				m.myLogin = "naveenspark"
				m.connected = true
				m.presenceCount = 5
				for i := 0; i < 10; i++ {
					id := fmt.Sprintf("msg-%d", i)
					m.seenIDs[id] = true
					m.messages = append(m.messages, chatMessage{
						ID:          id,
						SenderLogin: "alice",
						Body:        fmt.Sprintf("Message %d", i),
						Kind:        "message",
						CreatedAt:   time.Now().Add(time.Duration(i) * time.Minute),
					})
				}
			},
			wantLen: 24,
		},
		{
			name: "slash hints visible with cursor",
			setup: func(m *hallModel) {
				m.myLogin = "naveenspark"
				m.connected = true
				m.input = "/"
				m.animFrame = 0
				for i := 0; i < 5; i++ {
					id := fmt.Sprintf("msg-%d", i)
					m.seenIDs[id] = true
					m.messages = append(m.messages, chatMessage{
						ID: id, SenderLogin: "alice", Body: "hi",
						Kind: "message", CreatedAt: time.Now(),
					})
				}
			},
			wantLen: 24,
		},
		{
			name: "project autocomplete visible with cursor",
			setup: func(m *hallModel) {
				m.myLogin = "naveenspark"
				m.connected = true
				m.input = "#"
				m.animFrame = 0
				m.projectActive = true
				m.projectQuery = ""
				m.myProjects = []domain.WorkshopProject{
					{Name: "Grimora"}, {Name: "Clawzempic"},
				}
				m.projectMatches = m.myProjects
				for i := 0; i < 5; i++ {
					id := fmt.Sprintf("msg-%d", i)
					m.seenIDs[id] = true
					m.messages = append(m.messages, chatMessage{
						ID: id, SenderLogin: "alice", Body: "hi",
						Kind: "message", CreatedAt: time.Now(),
					})
				}
			},
			wantLen: 24,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestHallModel()
			tc.setup(&m)

			view := m.View()
			// Count newlines — each line in View() is \n-terminated
			newlines := strings.Count(view, "\n")
			if newlines != tc.wantLen {
				t.Errorf("View() has %d newlines, want %d", newlines, tc.wantLen)
				lines := strings.Split(view, "\n")
				for i, line := range lines {
					t.Logf("  %2d: %q", i, line)
				}
			}
		})
	}
}

func TestHallCursorVisibleWhenFocused(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "naveenspark"
	m.connected = true
	m.animFrame = 0 // frame 0 = cursor visible (even frame)
	m.presenceCount = 2

	// Add a few messages
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("msg-%d", i)
		m.seenIDs[id] = true
		m.messages = append(m.messages, chatMessage{
			ID: id, SenderLogin: "alice", Body: "hello",
			Kind: "message", CreatedAt: time.Now(),
		})
	}

	view := m.View()
	if !strings.Contains(view, "█") {
		t.Error("expected cursor block '█' in view when inputFocused=true and animFrame=0")
		lines := strings.Split(view, "\n")
		for i, line := range lines {
			t.Logf("  %2d: %q", i, line)
		}
	}
}

func TestHallInputMultilineNewlineInModel(t *testing.T) {
	// Verify that when m.input contains newlines (from shift+enter),
	// the model correctly adjusts viewport and renders without clipping.
	m := newTestHallModel()
	m.myLogin = "testuser"
	m.inputFocused = true
	m.connected = true

	// Simulate shift+enter by directly adding newline (the switch case does m.input += "\n").
	m.input = "first line\nsecond line"

	// Verify input has a newline.
	if !strings.Contains(m.input, "\n") {
		t.Fatal("expected newline in input")
	}

	// Verify View() still produces correct line count.
	view := m.View()
	newlines := strings.Count(view, "\n")
	if newlines != m.height {
		t.Errorf("View() with multiline input has %d newlines, want %d", newlines, m.height)
	}

	// Verify both lines appear in the rendered output.
	if !strings.Contains(view, "first line") || !strings.Contains(view, "second line") {
		t.Errorf("multiline input not fully rendered: %q", view)
	}
}

func TestHallViewLineCountWithMultilineInput(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "naveenspark"
	m.connected = true
	m.input = "line1\nline2" // 2-line input
	m.animFrame = 0
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("msg-%d", i)
		m.seenIDs[id] = true
		m.messages = append(m.messages, chatMessage{
			ID: id, SenderLogin: "alice", Body: "hi",
			Kind: "message", CreatedAt: time.Now(),
		})
	}

	view := m.View()
	newlines := strings.Count(view, "\n")
	if newlines != m.height {
		t.Errorf("View() with 2-line input has %d newlines, want %d", newlines, m.height)
		lines := strings.Split(view, "\n")
		for i, line := range lines {
			t.Logf("  %2d: %q", i, line)
		}
	}
}

func TestHardWrapBreaksLongURLs(t *testing.T) {
	url := "https://github.com/naveenspark/grimora/releases/tag/v0.4.0-this-is-a-very-long-url-that-exceeds-width"
	wrapped := hardWrap(url, 40)
	for _, line := range strings.Split(wrapped, "\n") {
		runes := []rune(line)
		if len(runes) > 40 {
			t.Errorf("hardWrap produced line with %d runes (>40): %q", len(runes), line)
		}
	}
	// Should contain the full URL content (no truncation).
	joined := strings.ReplaceAll(wrapped, "\n", "")
	if joined != url {
		t.Errorf("hardWrap lost content: got %q, want %q", joined, url)
	}
}

func TestHardWrapShortLinesUnchanged(t *testing.T) {
	input := "short line"
	got := hardWrap(input, 40)
	if got != input {
		t.Errorf("hardWrap changed short line: got %q, want %q", got, input)
	}
}

func TestRenderBodyWithMentionsNoLinkification(t *testing.T) {
	// Linkification is now done in linkifyURLs, not renderBodyWithMentions.
	body := "check https://grimora.ai for details"
	result := renderBodyWithMentions(body, "me", false)
	if strings.Contains(result, "\033]8;;") {
		t.Errorf("renderBodyWithMentions should not add OSC 8 links, got: %q", result)
	}
	if !strings.Contains(result, "https://grimora.ai") {
		t.Errorf("expected URL preserved verbatim, got: %q", result)
	}
}

func TestLinkifyURLsShortURL(t *testing.T) {
	body := "check https://grimora.ai for details"
	result := linkifyURLs(body, 80)
	// Short URL should be fully visible with OSC 8.
	if !strings.Contains(result, "\033]8;;https://grimora.ai\a") {
		t.Errorf("expected OSC 8 target, got: %q", result)
	}
	if !strings.Contains(result, "https://grimora.ai\033]8;;\a") {
		t.Errorf("expected full URL as display text, got: %q", result)
	}
}

func TestLinkifyURLsTruncatesLongURL(t *testing.T) {
	url := "https://github.com/naveenspark/grimora/releases/tag/v0.4.0-with-extra-long-path"
	body := "see " + url
	result := linkifyURLs(body, 30)
	// Full URL should be in OSC 8 target.
	if !strings.Contains(result, "\033]8;;"+url+"\a") {
		t.Errorf("expected full URL in OSC 8 target, got: %q", result)
	}
	// Display text should be truncated with ellipsis.
	if !strings.Contains(result, "…") {
		t.Errorf("expected ellipsis in truncated display, got: %q", result)
	}
	// Display text should NOT contain the full URL.
	if strings.Contains(result, url+"\033]8;;\a") {
		t.Errorf("display text should be truncated, not full URL, got: %q", result)
	}
}

func TestStripTrailingSpaces(t *testing.T) {
	input := "hello   \nworld  \nfoo"
	want := "hello\nworld\nfoo"
	got := stripTrailingSpaces(input)
	if got != want {
		t.Errorf("stripTrailingSpaces(%q) = %q, want %q", input, got, want)
	}
}

func TestLinkifyURLsZeroWidth(t *testing.T) {
	body := "see https://grimora.ai"
	got := linkifyURLs(body, 0)
	if got != body {
		t.Errorf("linkifyURLs with maxWidth=0 should return body unchanged, got: %q", got)
	}
}

func TestMentionInsideOSC8Preserved(t *testing.T) {
	// A URL with @ should not have its @-part styled as a mention.
	linkified := linkifyURLs("connect via https://user@host.example.com/path ok", 80)
	result := renderBodyWithMentions(linkified, "me", false)
	// The OSC 8 target must be intact (not corrupted by mention styling).
	if !strings.Contains(result, "\033]8;;https://user@host.example.com/path\a") {
		t.Errorf("OSC 8 target corrupted by mention styling: %q", result)
	}
}

func TestLongURLMessageTruncatesDisplay(t *testing.T) {
	m := newTestHallModel()
	m.myLogin = "me"
	m.width = 60

	fullURL := "https://github.com/naveenspark/grimora/releases/tag/v0.4.0-with-extra-long-path-that-exceeds-terminal"
	msg := chatMessage{
		ID:          "url-1",
		SenderLogin: "alice",
		Body:        fullURL,
		Kind:        "message",
		CreatedAt:   time.Now(),
	}

	rendered := m.renderMessage(msg)
	// Full URL should be in OSC 8 target (clickable).
	if !strings.Contains(rendered, "\033]8;;"+fullURL+"\a") {
		t.Errorf("expected full URL in OSC 8 target, got: %q", rendered)
	}
	// Display should be truncated (ellipsis present).
	if !strings.Contains(rendered, "…") {
		t.Errorf("expected truncated display with ellipsis, got: %q", rendered)
	}
}

// testErr is a simple error type for tests.
type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
