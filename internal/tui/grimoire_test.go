package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/naveenspark/grimora/pkg/domain"
)

func newTestGrimoireModel() grimoireModel {
	m := newGrimoireModel(nil)
	m.width = 80
	m.height = 24
	m.loading = false
	return m
}

func makeTestSpell(text, tag string) domain.Spell {
	return domain.Spell{
		ID:        uuid.New(),
		Text:      text,
		Tag:       tag,
		Potency:   2,
		Upvotes:   5,
		CreatedAt: time.Now(),
		Author: &domain.Author{
			Login:   "testauthor",
			GuildID: "loomari",
		},
	}
}

func makeTestWeapon(name string) domain.Weapon {
	return domain.Weapon{
		ID:             uuid.New(),
		Name:           name,
		Description:    "A test weapon description",
		RepositoryURL:  "https://github.com/test/repo",
		GitHubStars:    1500,
		GitHubLanguage: "Go",
		CreatedAt:      time.Now(),
	}
}

func TestGrimoireSpellListRendersTitles(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{
		makeTestSpell("Debug with a rubber duck first", "debugging"),
		makeTestSpell("Refactor in small increments", "refactoring"),
	}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	view := m.View()
	if !strings.Contains(view, "Debug with a rubber duck first") {
		t.Errorf("expected spell text in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Refactor in small increments") {
		t.Errorf("expected second spell text in view, got:\n%s", view)
	}
}

func TestGrimoireWeaponListRendersNames(t *testing.T) {
	m := newTestGrimoireModel()
	m.mode = grimoireModeWeapons
	weapons := []domain.Weapon{
		makeTestWeapon("ripgrep"),
		makeTestWeapon("fzf"),
	}
	m, _ = m.Update(weaponsLoadedMsg{weapons: weapons})

	view := m.View()
	if !strings.Contains(view, "ripgrep") {
		t.Errorf("expected 'ripgrep' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "fzf") {
		t.Errorf("expected 'fzf' in view, got:\n%s", view)
	}
}

func TestGrimoireModeToggleSwitchesBetweenSpellsAndWeapons(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("Test spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	// Initially spells mode
	view := m.View()
	if !strings.Contains(view, "spells") {
		t.Errorf("expected 'spells' in view before toggle, got:\n%s", view)
	}

	// Toggle to weapons
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	m.loading = false
	weapons := []domain.Weapon{makeTestWeapon("some-tool")}
	m, _ = m.Update(weaponsLoadedMsg{weapons: weapons})

	view = m.View()
	if !strings.Contains(view, "weapons") {
		t.Errorf("expected 'weapons' in view after toggle, got:\n%s", view)
	}
}

func TestGrimoireSearchModeActivatesOnSlash(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("some spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})

	if !m.editing {
		t.Error("expected editing=true after '/' key, got false")
	}

	view := m.View()
	if !strings.Contains(view, "/") {
		t.Errorf("expected '/' in view when searching, got:\n%s", view)
	}
}

func TestGrimoireSortCycling(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	// Initial sort is "new"
	if m.sortBy != "new" {
		t.Errorf("expected initial sortBy='new', got %q", m.sortBy)
	}

	// Press 's' -> top
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m.loading = false
	m, _ = m.Update(spellsLoadedMsg{spells: spells})
	if m.sortBy != "top" {
		t.Errorf("expected sortBy='top' after first 's', got %q", m.sortBy)
	}

	// Press 's' again -> casts
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m.loading = false
	m, _ = m.Update(spellsLoadedMsg{spells: spells})
	if m.sortBy != "casts" {
		t.Errorf("expected sortBy='casts' after second 's', got %q", m.sortBy)
	}

	// Press 's' again -> new (wraps)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m.loading = false
	m, _ = m.Update(spellsLoadedMsg{spells: spells})
	if m.sortBy != "new" {
		t.Errorf("expected sortBy='new' after third 's', got %q", m.sortBy)
	}
}

func TestGrimoireDetailViewShowsSpellText(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{
		makeTestSpell("You are an expert debugger. Analyze the stack trace carefully.", "debugging"),
	}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	// Enter detail view
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := m.View()
	if !m.detail {
		t.Error("expected detail=true after Enter, got false")
	}
	if !strings.Contains(view, "You are an expert debugger") {
		t.Errorf("expected spell text in detail view, got:\n%s", view)
	}
	if !strings.Contains(view, "back") {
		t.Errorf("expected 'back' hint in detail view, got:\n%s", view)
	}
}

func TestGrimoireUpvoteSendsCommand(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("A great debugging spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	// Upvote returns a cmd (even if we can't execute it without a real client)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd == nil {
		t.Error("expected upvote to return a command, got nil")
	}
}

func TestGrimoireCopySendsCommand(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("Copy this spell text", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if cmd == nil {
		t.Error("expected copy to return a command, got nil")
	}
}

func TestGrimoireBackEscExitsDetail(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("Some spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	// Enter detail
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.detail {
		t.Fatal("expected to be in detail view")
	}

	// Press Esc to go back
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.detail {
		t.Error("expected detail=false after Esc, got true")
	}
}

func TestGrimoireTagCyclesOnT(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	// Initially no tag filter (= "all")
	if m.tagFilter != "" {
		t.Errorf("expected empty tagFilter initially, got %q", m.tagFilter)
	}

	// The 't' key cycles to the next tag and triggers a reload
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})

	if m.tagFilter == "" {
		t.Error("expected tagFilter to change after 't' key, got empty (still 'all')")
	}
	if cmd == nil {
		t.Error("expected tag cycle to return a reload command, got nil")
	}
}

func TestGrimoireInlineTagBarRendered(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	view := m.View()
	if !strings.Contains(view, "debugging") {
		t.Errorf("expected 'debugging' in inline tag bar, got:\n%s", view)
	}
	if !strings.Contains(view, "database") {
		t.Errorf("expected 'database' in inline tag bar, got:\n%s", view)
	}
	if !strings.Contains(view, "observability") {
		t.Errorf("expected 'observability' in inline tag bar, got:\n%s", view)
	}
}

func TestGrimoireEmptySpellListShowsNoSpellsFound(t *testing.T) {
	m := newTestGrimoireModel()
	m, _ = m.Update(spellsLoadedMsg{spells: []domain.Spell{}})

	view := m.View()
	if !strings.Contains(view, "no spells found") {
		t.Errorf("expected 'no spells found' in view, got:\n%s", view)
	}
}

func TestGrimoireEmptyWeaponListShowsNoWeaponsFound(t *testing.T) {
	m := newTestGrimoireModel()
	m.mode = grimoireModeWeapons
	m, _ = m.Update(weaponsLoadedMsg{weapons: []domain.Weapon{}})

	view := m.View()
	if !strings.Contains(view, "no weapons found") {
		t.Errorf("expected 'no weapons found' in view, got:\n%s", view)
	}
}

func TestGrimoireNavigation(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{
		makeTestSpell("Spell one", "debugging"),
		makeTestSpell("Spell two", "testing"),
		makeTestSpell("Spell three", "architecture"),
	}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	if m.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", m.cursor)
	}

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("expected cursor=1 after j, got %d", m.cursor)
	}

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("expected cursor=0 after k, got %d", m.cursor)
	}
}

func TestGrimoireUpvoteResultSetsStatusMsg(t *testing.T) {
	m := newTestGrimoireModel()
	spells := []domain.Spell{makeTestSpell("spell", "debugging")}
	m, _ = m.Update(spellsLoadedMsg{spells: spells})

	// Simulate a successful upvote result (no real API call)
	m, _ = m.Update(upvoteResultMsg{err: nil})
	// After success, the model reloads spells (cmd will be non-nil but statusMsg set)
	if m.statusMsg != "upvoted!" {
		t.Errorf("expected statusMsg='upvoted!', got %q", m.statusMsg)
	}
}
