package tui

import (
	"strings"
	"testing"

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
		{"1", viewHome},
		{"2", viewHall},
		{"3", viewGrimoire},
		{"4", viewYou},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			model, _ := newTestApp().Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
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
	// Default view is home, not editing — q should quit
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected quit command on 'q', got nil")
	}
	// tea.Quit returns a specific command; we verify it's non-nil
}

func TestAppEscFromCreateReturnsToHome(t *testing.T) {
	a := newTestApp()

	// Switch to create view
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	a = model.(App)
	if a.view != viewCreate {
		t.Fatalf("expected viewCreate after 'n', got %d", a.view)
	}

	// Press Esc to go back home
	model, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = model.(App)
	if a.view != viewHome {
		t.Errorf("expected viewHome after Esc from create, got %d", a.view)
	}
}

func TestAppIsEditingGrimoireSearch(t *testing.T) {
	a := newTestApp()

	// Switch to grimoire tab
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
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

func TestAppViewRendersTabBar(t *testing.T) {
	a := newTestApp()
	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	a = model.(App)

	view := a.View()
	if !strings.Contains(view, "Home") {
		t.Errorf("expected 'Home' tab in app view, got:\n%s", view)
	}
	if !strings.Contains(view, "Hall") {
		t.Errorf("expected 'Hall' tab in app view, got:\n%s", view)
	}
	if !strings.Contains(view, "Grimoire") {
		t.Errorf("expected 'Grimoire' tab in app view, got:\n%s", view)
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

// Re-use makeTestMagicianCard from peek_test.go (same package)
// It's already defined there.
