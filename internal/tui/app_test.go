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

func newTestApp() App {
	a := NewApp(nil)
	a.width = 80
	a.height = 30
	return a
}

func TestAppTabSwitching(t *testing.T) {
	tests := []struct {
		key      string
		wantView view
	}{
		{"1", viewHall},
		{"2", viewGrimoire},
		{"3", viewThreads},
		{"4", viewBoard},
		{"5", viewYou},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			app := newTestApp()
			app.hall.inputFocused = false // nav mode so global keys work
			model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			a := model.(App)
			if a.view != tc.wantView {
				t.Errorf("after key %q: expected view=%d, got %d", tc.key, tc.wantView, a.view)
			}
		})
	}
}

func TestAppPeekOverlayOpenAndClose(t *testing.T) {
	a := newTestApp()

	// Open peek via showPeekMsg
	model, _ := a.Update(showPeekMsg{login: "someuser"})
	a = model.(App)
	if !a.peekOpen {
		t.Fatal("expected peekOpen=true after showPeekMsg, got false")
	}

	// Close with Esc (peek captures the key)
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = model.(App)
	if a.peekOpen {
		t.Error("expected peekOpen=false after Esc in peek, got true")
	}
}

func TestAppGlobalQuitOnQ(t *testing.T) {
	a := newTestApp()
	a.hall.inputFocused = false // nav mode so global keys work
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected quit command on 'q', got nil")
	}
}

func TestAppEscFromCreateReturnsToHall(t *testing.T) {
	a := newTestApp()
	a.hall.inputFocused = false // nav mode so global keys work

	// Switch to create view
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	a = model.(App)
	if a.view != viewCreate {
		t.Fatalf("expected viewCreate after 'n', got %d", a.view)
	}

	// Press Esc to go back to Hall
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = model.(App)
	if a.view != viewHall {
		t.Errorf("expected viewHall after Esc from create, got %d", a.view)
	}
}

func TestAppIsEditingGrimoireSearch(t *testing.T) {
	a := newTestApp()
	a.hall.inputFocused = false // nav mode so global keys work

	// Switch to grimoire tab
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	a = model.(App)
	a.grimoire.loading = false
	a.grimoire.spells = nil

	// Start search
	a.grimoire.editing = true
	if !a.isEditing() {
		t.Error("expected isEditing=true when grimoire.editing=true")
	}
}

func TestAppIsEditingCreate(t *testing.T) {
	a := newTestApp()
	a.view = viewCreate
	if !a.isEditing() {
		t.Error("expected isEditing=true when view=viewCreate")
	}
}

func TestAppIsEditingHallInput(t *testing.T) {
	a := newTestApp()
	a.view = viewHall
	a.hall.inputFocused = true
	if !a.isEditing() {
		t.Error("expected isEditing=true when hall.inputFocused=true")
	}

	a.hall.inputFocused = false
	if a.isEditing() {
		t.Error("expected isEditing=false when hall.inputFocused=false and view=viewHall")
	}
}

func TestAppIsEditingYouWorkshop(t *testing.T) {
	a := newTestApp()
	a.view = viewYou

	// wsNormal should not be editing
	a.you.wsState = wsNormal
	if a.isEditing() {
		t.Error("expected isEditing=false when you.wsState=wsNormal")
	}

	// wsEditing should be editing
	a.you.wsState = wsEditing
	if !a.isEditing() {
		t.Error("expected isEditing=true when you.wsState=wsEditing")
	}

	// wsAdding should be editing
	a.you.wsState = wsAdding
	if !a.isEditing() {
		t.Error("expected isEditing=true when you.wsState=wsAdding")
	}

	// wsDeleting should be editing
	a.you.wsState = wsDeleting
	if !a.isEditing() {
		t.Error("expected isEditing=true when you.wsState=wsDeleting")
	}
}

func TestAppIsEditingThreadsInput(t *testing.T) {
	a := newTestApp()
	a.view = viewThreads

	a.threads.inputFocused = false
	if a.isEditing() {
		t.Error("expected isEditing=false when threads.inputFocused=false")
	}

	a.threads.inputFocused = true
	if !a.isEditing() {
		t.Error("expected isEditing=true when threads.inputFocused=true")
	}
}

func TestAppViewRendersTabBar(t *testing.T) {
	a := newTestApp()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a = model.(App)

	view := a.View()
	if !strings.Contains(view, "Hall") {
		t.Errorf("expected 'Hall' tab in app view, got:\n%s", view)
	}
	if !strings.Contains(view, "Grimoire") {
		t.Errorf("expected 'Grimoire' tab in app view, got:\n%s", view)
	}
	if !strings.Contains(view, "Threads") {
		t.Errorf("expected 'Threads' tab in app view, got:\n%s", view)
	}
	if !strings.Contains(view, "Board") {
		t.Errorf("expected 'Board' tab in app view, got:\n%s", view)
	}
	if !strings.Contains(view, "You") {
		t.Errorf("expected 'You' tab in app view, got:\n%s", view)
	}
}

func TestAppMeLoadedPropagatesIdentity(t *testing.T) {
	a := newTestApp()
	me := &domain.Magician{
		ID:          uuid.New(),
		GitHubLogin: "rootmage",
		GuildID:     "cipher",
		CardNumber:  1,
	}

	model, _ := a.Update(meLoadedMsg{me: me})
	a = model.(App)

	if a.me == nil {
		t.Fatal("expected a.me to be set after meLoadedMsg")
	}
	if a.me.GitHubLogin != "rootmage" {
		t.Errorf("expected me.GitHubLogin='rootmage', got %q", a.me.GitHubLogin)
	}
	// Should also propagate to hall
	if a.hall.myLogin != "rootmage" {
		t.Errorf("expected hall.myLogin='rootmage', got %q", a.hall.myLogin)
	}
}

func TestAppPeekOverlayRendersInView(t *testing.T) {
	a := newTestApp()
	sizedModel, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a = sizedModel.(App)

	// Open peek overlay
	model, _ := a.Update(showPeekMsg{login: "peekeduser"})
	a = model.(App)

	// Load card data
	card := makeTestMagicianCard("peekeduser", "amarok", false)
	a.peek, _ = a.peek.Update(peekLoadedMsg{card: card})

	view := a.View()
	if !strings.Contains(view, "peekeduser") {
		t.Errorf("expected peeked user name in app view with open overlay, got:\n%s", view)
	}
}

func TestAppShimmerFrameIncrements(t *testing.T) {
	a := newTestApp()
	initial := a.frame

	model, _ := a.Update(shimmerTickMsg{})
	a = model.(App)

	if a.frame != initial+1 {
		t.Errorf("expected frame=%d after shimmerTickMsg, got %d", initial+1, a.frame)
	}
}

func TestAppQNotFiredWhenEditing(t *testing.T) {
	a := newTestApp()
	// Put app in grimoire search (editing) mode
	a.view = viewGrimoire
	a.grimoire.editing = true
	a.grimoire.loading = false

	// 'q' while editing should NOT quit — it should be passed to grimoire search
	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	a = model.(App)
	// cmd should not be tea.Quit
	_ = cmd
	if a.grimoire.search != "q" {
		t.Errorf("expected grimoire.search to be 'q', got %q", a.grimoire.search)
	}
}

func TestAppHallLayoutFitsTerminal(t *testing.T) {
	termHeight := 30
	a := newTestApp()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: termHeight})
	a = model.(App)

	// Simulate logged-in user with messages
	a.me = &domain.Magician{
		ID:          uuid.New(),
		GitHubLogin: "naveenspark",
		GuildID:     "cipher",
		CardNumber:  42,
	}
	a.hall.myLogin = "naveenspark"
	a.hall.connected = true
	a.hall.presenceCount = 3
	a.hall.animFrame = 0

	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("msg-%d", i)
		a.hall.messages = append(a.hall.messages, chatMessage{
			ID: id, SenderLogin: "alice", Body: fmt.Sprintf("Message %d", i),
			Kind: "message", CreatedAt: time.Now(),
		})
		a.hall.seenIDs[id] = true
	}

	view := a.View()
	lines := strings.Split(view, "\n")
	// View may end with trailing \n → last element is ""
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) != termHeight {
		t.Errorf("App.View() has %d lines, want %d (terminal height)", len(lines), termHeight)
		for i, line := range lines {
			t.Logf("  %2d: %q", i, line)
		}
	}

	// Cursor must be visible
	if !strings.Contains(view, "█") {
		t.Error("expected cursor '█' in full app view")
	}

	// Input line must show the user's login in the input area
	if !strings.Contains(view, "naveenspark") {
		t.Error("expected 'naveenspark' in full app view (input line)")
	}
}

func TestAppHallLayoutConnectingState(t *testing.T) {
	termHeight := 30
	a := newTestApp()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: termHeight})
	a = model.(App)
	a.hall.animFrame = 0

	view := a.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) != termHeight {
		t.Errorf("App.View() connecting state has %d lines, want %d", len(lines), termHeight)
		for i, line := range lines {
			t.Logf("  %2d: %q", i, line)
		}
	}

	if !strings.Contains(view, "connecting") {
		t.Error("expected 'connecting' text in app view")
	}
}

// Re-use makeTestMagicianCard from peek_test.go (same package)
// It's already defined there.
