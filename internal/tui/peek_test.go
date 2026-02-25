package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/naveenspark/grimora/pkg/domain"
)

func newTestPeekModel() peekModel {
	return newPeekModel(nil)
}

func makeTestMagicianCard(login, guildID string, online bool) *domain.MagicianCard {
	return &domain.MagicianCard{
		Magician: domain.Magician{
			ID:          uuid.New(),
			GitHubLogin: login,
			GuildID:     guildID,
			CardNumber:  7,
		},
		SpellCount:   10,
		TotalPotency: 2,
		Online:       online,
	}
}

func TestPeekLoadSuccessShowsCard(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("wizarduser", "cipher", false)
	m, _ = m.Update(peekLoadedMsg{card: card})

	view := m.View()
	if !strings.Contains(view, "wizarduser") {
		t.Errorf("expected 'wizarduser' in peek view, got:\n%s", view)
	}
}

func TestPeekLoadErrorShowsError(t *testing.T) {
	m := newTestPeekModel()
	m, _ = m.Update(peekLoadedMsg{err: errors.New("user not found")})

	view := m.View()
	if !strings.Contains(view, "error") {
		t.Errorf("expected 'error' in peek view on error, got:\n%s", view)
	}
	if !strings.Contains(view, "user not found") {
		t.Errorf("expected error message in peek view, got:\n%s", view)
	}
}

func TestPeekFollowToggle(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("targetuser", "loomari", true)
	card.IsFollowing = false
	m, _ = m.Update(peekLoadedMsg{card: card})

	// Simulate successful follow
	m, _ = m.Update(peekFollowMsg{login: "targetuser", err: nil})

	if !m.card.IsFollowing {
		t.Error("expected IsFollowing=true after follow, got false")
	}

	// Toggle again (unfollow)
	m, _ = m.Update(peekFollowMsg{login: "targetuser", err: nil})
	if m.card.IsFollowing {
		t.Error("expected IsFollowing=false after unfollow, got true")
	}
}

func TestPeekCloseOnEsc(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("someuser", "nyx", false)
	m, _ = m.Update(peekLoadedMsg{card: card})

	if m.closed {
		t.Fatal("peek should not be closed initially")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !m.closed {
		t.Error("expected closed=true after Esc, got false")
	}
}

func TestPeekCloseOnQ(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("someuser", "fathom", false)
	m, _ = m.Update(peekLoadedMsg{card: card})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if !m.closed {
		t.Error("expected closed=true after 'q', got false")
	}
}

func TestPeekEmblemDisplayed(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("emblemuser", "loomari", false)
	m, _ = m.Update(peekLoadedMsg{card: card})

	view := m.View()
	// loomari = spider emoji
	if !strings.Contains(view, GuildEmblem("loomari")) {
		t.Errorf("expected loomari emblem in peek view, got:\n%s", view)
	}
}

func TestPeekOnlineStatusDisplayed(t *testing.T) {
	t.Run("online user", func(t *testing.T) {
		m := newTestPeekModel()
		card := makeTestMagicianCard("onlinewizard", "cipher", true)
		m, _ = m.Update(peekLoadedMsg{card: card})

		view := m.View()
		if !strings.Contains(view, "online") {
			t.Errorf("expected 'online' in peek view for online user, got:\n%s", view)
		}
	})

	t.Run("offline user", func(t *testing.T) {
		m := newTestPeekModel()
		card := makeTestMagicianCard("offlinewizard", "amarok", false)
		m, _ = m.Update(peekLoadedMsg{card: card})

		view := m.View()
		if !strings.Contains(view, "offline") {
			t.Errorf("expected 'offline' in peek view for offline user, got:\n%s", view)
		}
	})
}

func TestPeekWorkshopProjectsDisplayed(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("projectwizard", "fathom", false)
	m, _ = m.Update(peekLoadedMsg{card: card})

	projects := []domain.WorkshopProject{
		{
			ID:        uuid.New(),
			Name:      "Arcane Compiler",
			Insight:   "Compiles spells to machine code",
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Name:      "Mana Router",
			Insight:   "Routes mana efficiently",
			UpdatedAt: time.Now(),
		},
	}
	m, _ = m.Update(peekWorkshopMsg{projects: projects})

	view := m.View()
	if !strings.Contains(view, "Arcane Compiler") {
		t.Errorf("expected 'Arcane Compiler' in peek view, got:\n%s", view)
	}
	if !strings.Contains(view, "Mana Router") {
		t.Errorf("expected 'Mana Router' in peek view, got:\n%s", view)
	}
	if !strings.Contains(view, "WORKSHOP") {
		t.Errorf("expected 'WORKSHOP' section header in peek view, got:\n%s", view)
	}
}

func TestPeekFollowKeySendsCommand(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("followtarget", "loomari", false)
	m, _ = m.Update(peekLoadedMsg{card: card})

	// Press 'f' — should return a command (nil client means it will panic if executed,
	// but we only check the cmd is non-nil to verify the key was handled)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	if cmd == nil {
		t.Error("expected follow command returned on 'f' key, got nil")
	}
}

func TestPeekLoadingStateBeforeCard(t *testing.T) {
	m := newTestPeekModel()
	// No peekLoadedMsg sent — card is nil
	view := m.View()
	if !strings.Contains(view, "loading") {
		t.Errorf("expected 'loading' in peek view before card loads, got:\n%s", view)
	}
}

func TestPeekStatsDisplayed(t *testing.T) {
	m := newTestPeekModel()
	card := makeTestMagicianCard("statswizard", "cipher", false)
	card.SpellCount = 42
	card.TotalPotency = 3
	card.CardNumber = 5
	m, _ = m.Update(peekLoadedMsg{card: card})

	view := m.View()
	if !strings.Contains(view, "42") {
		t.Errorf("expected spell count '42' in peek stats, got:\n%s", view)
	}
}
