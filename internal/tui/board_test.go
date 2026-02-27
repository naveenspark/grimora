package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/naveenspark/grimora/pkg/domain"
)

func newTestBoardModel() boardModel {
	m := newBoardModel(nil)
	m.width = 80
	m.height = 30
	return m
}

func makeTestLeaderboardEntry(rank int, login, guild string, spells, potency int) domain.LeaderboardEntry {
	return domain.LeaderboardEntry{
		Rank:         rank,
		Login:        login,
		GuildID:      guild,
		SpellsForged: spells,
		TotalPotency: potency,
	}
}

func TestBoardRendersEntries(t *testing.T) {
	m := newTestBoardModel()
	entries := []domain.LeaderboardEntry{
		makeTestLeaderboardEntry(1, "topwizard", "cipher", 20, 10),
		makeTestLeaderboardEntry(2, "secondmage", "loomari", 15, 7),
	}
	m, _ = m.Update(boardLoadedMsg{entries: entries})

	view := m.View()
	if !strings.Contains(view, "topwizard") {
		t.Errorf("expected 'topwizard' in board view, got:\n%s", view)
	}
	if !strings.Contains(view, "secondmage") {
		t.Errorf("expected 'secondmage' in board view, got:\n%s", view)
	}
}

func TestBoardHighlightsSelf(t *testing.T) {
	m := newTestBoardModel()
	m.myLogin = "itsme"
	entries := []domain.LeaderboardEntry{
		makeTestLeaderboardEntry(1, "itsme", "nyx", 10, 5),
		makeTestLeaderboardEntry(2, "other", "cipher", 8, 3),
	}
	m, _ = m.Update(boardLoadedMsg{entries: entries})

	view := m.View()
	if !strings.Contains(view, "<- you") {
		t.Errorf("expected '<- you' marker for self in board view, got:\n%s", view)
	}
}

func TestBoardGuildFilterCycles(t *testing.T) {
	m := newTestBoardModel()
	m, _ = m.Update(boardLoadedMsg{entries: []domain.LeaderboardEntry{
		makeTestLeaderboardEntry(1, "wizard", "cipher", 10, 5),
	}})

	// Initial: no filter
	if m.guildFilter != "" {
		t.Errorf("expected empty guild filter initially, got %q", m.guildFilter)
	}

	// Press 'g' to cycle to first guild
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.guildFilter != "loomari" {
		t.Errorf("expected guild filter 'loomari' after first g, got %q", m.guildFilter)
	}
	if cmd == nil {
		t.Error("expected reload command after guild filter change, got nil")
	}
}

func TestBoardEmptyState(t *testing.T) {
	m := newTestBoardModel()
	m, _ = m.Update(boardLoadedMsg{entries: nil})

	view := m.View()
	if !strings.Contains(view, "no magicians yet") {
		t.Errorf("expected 'no magicians yet' in empty board, got:\n%s", view)
	}
}

func TestBoardPeekTriggered(t *testing.T) {
	m := newTestBoardModel()
	entries := []domain.LeaderboardEntry{
		makeTestLeaderboardEntry(1, "peekme", "cipher", 5, 2),
	}
	m, _ = m.Update(boardLoadedMsg{entries: entries})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if cmd == nil {
		t.Error("expected peek command on 'p', got nil")
	}
}

func TestBoardMeLoadedSetsMyLogin(t *testing.T) {
	m := newTestBoardModel()
	me := &domain.Magician{
		ID:          uuid.New(),
		GitHubLogin: "mymage",
	}
	m, _ = m.Update(meLoadedMsg{me: me})
	if m.myLogin != "mymage" {
		t.Errorf("expected myLogin='mymage', got %q", m.myLogin)
	}
}

func TestBoardCityFilterCycles(t *testing.T) {
	m := newTestBoardModel()
	entries := []domain.LeaderboardEntry{
		{Rank: 1, Login: "alice", GuildID: "cipher", City: "Seattle", SpellsForged: 10},
		{Rank: 2, Login: "bob", GuildID: "nyx", City: "Berlin", SpellsForged: 8},
	}
	m, _ = m.Update(boardLoadedMsg{entries: entries})

	if m.cityFilter != "" {
		t.Fatalf("expected empty city filter initially, got %q", m.cityFilter)
	}

	// First press: should cycle to first city alphabetically (Berlin)
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if m.cityFilter != "Berlin" {
		t.Errorf("expected city filter 'Berlin' after first c, got %q", m.cityFilter)
	}
	if cmd == nil {
		t.Error("expected reload command after city filter change")
	}
}

func TestBoardCityOrderDerived(t *testing.T) {
	m := newTestBoardModel()
	entries := []domain.LeaderboardEntry{
		{Rank: 1, Login: "a", City: "Zurich"},
		{Rank: 2, Login: "b", City: "Amsterdam"},
		{Rank: 3, Login: "c", City: "Zurich"}, // duplicate
		{Rank: 4, Login: "d", City: ""},       // no city
	}
	m, _ = m.Update(boardLoadedMsg{entries: entries})

	// cityOrder should be ["", "Amsterdam", "Zurich"]
	if len(m.cityOrder) != 3 {
		t.Fatalf("expected 3 city options (all + 2 unique), got %d: %v", len(m.cityOrder), m.cityOrder)
	}
	if m.cityOrder[0] != "" || m.cityOrder[1] != "Amsterdam" || m.cityOrder[2] != "Zurich" {
		t.Errorf("unexpected city order: %v", m.cityOrder)
	}
}

func TestBoardCityFilterInHeader(t *testing.T) {
	m := newTestBoardModel()
	m.cityFilter = "Tokyo"
	m.entries = []domain.LeaderboardEntry{
		{Rank: 1, Login: "x", City: "Tokyo", SpellsForged: 5},
	}

	view := m.View()
	if !strings.Contains(view, "Tokyo") {
		t.Errorf("expected 'Tokyo' in board header, got:\n%s", view)
	}
}
