package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/naveenspark/grimora/pkg/domain"
)

func newTestHomeModel() homeModel {
	m := newHomeModel(nil)
	m.width = 80
	m.height = 24
	return m
}

func makeEvent(kind, login, title, tag, voice string) domain.StreamEvent {
	return domain.StreamEvent{
		Kind:          kind,
		ID:            uuid.New(),
		MagicianLogin: login,
		Title:         title,
		Tag:           tag,
		Voice:         voice,
		CreatedAt:     time.Now(),
	}
}

func TestHomeStreamLoaded(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		makeEvent("spell", "testuser", "Test spell title", "debugging", ""),
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "testuser") {
		t.Errorf("expected view to contain 'testuser', got:\n%s", view)
	}
}

func TestHomeStreamLoadedWithError(t *testing.T) {
	m := newTestHomeModel()
	m, _ = m.Update(streamLoadedMsg{err: errors.New("connection refused")})

	view := m.View()
	if !strings.Contains(view, "error") {
		t.Errorf("expected view to contain 'error', got:\n%s", view)
	}
	if !strings.Contains(view, "connection refused") {
		t.Errorf("expected view to contain 'connection refused', got:\n%s", view)
	}
}

func TestHomeEmptyStreamShowsNoActivity(t *testing.T) {
	m := newTestHomeModel()
	m, _ = m.Update(streamLoadedMsg{events: nil})

	view := m.View()
	if !strings.Contains(view, "no activity yet") {
		t.Errorf("expected 'no activity yet', got:\n%s", view)
	}
}

func TestHomeForgeEventSpellKind(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		makeEvent("spell", "alice", "My debugging spell", "debugging", ""),
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "alice") {
		t.Errorf("expected view to contain author 'alice', got:\n%s", view)
	}
	if !strings.Contains(view, "forged") {
		t.Errorf("expected view to contain 'forged' action, got:\n%s", view)
	}
}

func TestHomeCastEventWeaponKind(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		makeEvent("weapon", "", "Awesome Tool v2", "", ""),
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "Awesome Tool v2") {
		t.Errorf("expected view to contain weapon title, got:\n%s", view)
	}
	if !strings.Contains(view, "cast") {
		t.Errorf("expected view to contain 'cast', got:\n%s", view)
	}
}

func TestHomeJoinEventMemberKind(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		{
			Kind:          "member",
			ID:            uuid.New(),
			MagicianLogin: "newmage",
			GuildID:       "loomari",
			CreatedAt:     time.Now(),
		},
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "newmage") {
		t.Errorf("expected view to contain 'newmage', got:\n%s", view)
	}
	if !strings.Contains(view, "joined") {
		t.Errorf("expected view to contain 'joined', got:\n%s", view)
	}
}

func TestHomeRejectEvent(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		makeEvent("reject", "", "Weak spell attempt", "", "The grimoire found this lacking."),
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "rejected") {
		t.Errorf("expected view to contain 'rejected', got:\n%s", view)
	}
	if !strings.Contains(view, "Weak spell attempt") {
		t.Errorf("expected view to contain rejected title, got:\n%s", view)
	}
}

func TestHomeMuseEventRendersGrimoireVoice(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		{
			Kind:      "muse",
			ID:        uuid.New(),
			Voice:     "The patterns speak to those who listen.",
			CreatedAt: time.Now(),
		},
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "Grimoire") {
		t.Errorf("expected view to contain 'Grimoire' label, got:\n%s", view)
	}
	if !strings.Contains(view, "The patterns speak to those who listen.") {
		t.Errorf("expected view to contain muse voice, got:\n%s", view)
	}
}

func TestHomeFeaturedEventRendersPurpleStyle(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		{
			Kind:          "featured",
			ID:            uuid.New(),
			MagicianLogin: "starbuilder",
			Title:         "Distributed Spell Engine",
			Voice:         "A truly innovative approach",
			Tag:           "architecture",
			CreatedAt:     time.Now(),
		},
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "starbuilder") {
		t.Errorf("expected view to contain 'starbuilder', got:\n%s", view)
	}
	if !strings.Contains(view, "Distributed Spell Engine") {
		t.Errorf("expected view to contain project title, got:\n%s", view)
	}
	if !strings.Contains(view, "building") {
		t.Errorf("expected view to contain 'building', got:\n%s", view)
	}
}

func TestHomeConvoEventRendersUserQuestionAndResponse(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		{
			Kind:      "convo",
			ID:        uuid.New(),
			Title:     "How do I debug a memory leak?",
			Voice:     "Begin by profiling allocations with a heap tool.",
			CreatedAt: time.Now(),
		},
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "How do I debug a memory leak?") {
		t.Errorf("expected view to contain user question, got:\n%s", view)
	}
	if !strings.Contains(view, "Begin by profiling") {
		t.Errorf("expected view to contain grimoire response, got:\n%s", view)
	}
}

func TestHomeVoiceLineAppendedWhenVoiceFieldNonEmpty(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		{
			Kind:          "spell",
			ID:            uuid.New(),
			MagicianLogin: "mage",
			Title:         "Debug spell",
			Tag:           "debugging",
			Voice:         "An elegant incantation for hunting ghosts.",
			CreatedAt:     time.Now(),
		},
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "An elegant incantation for hunting ghosts.") {
		t.Errorf("expected view to contain voice line, got:\n%s", view)
	}
}

func TestHomeAutoRefreshTickTriggersReload(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{makeEvent("spell", "user1", "spell1", "debugging", "")}
	m, _ = m.Update(streamLoadedMsg{events: events})

	// Sending a streamTickMsg should trigger a load (returns a non-nil cmd)
	_, cmd := m.Update(streamTickMsg(time.Now()))
	if cmd == nil {
		t.Error("expected streamTickMsg to return a load command, got nil")
	}
}

func TestHomeLegendLineRendered(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{makeEvent("spell", "user", "title", "debugging", "")}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "forge") {
		t.Errorf("expected legend to contain 'forge', got:\n%s", view)
	}
	if !strings.Contains(view, "cast") {
		t.Errorf("expected legend to contain 'cast', got:\n%s", view)
	}
}

func TestHomeTagChipRenderedForSpell(t *testing.T) {
	m := newTestHomeModel()
	events := []domain.StreamEvent{
		makeEvent("spell", "user", "A debugging spell", "debugging", ""),
	}
	m, _ = m.Update(streamLoadedMsg{events: events})

	view := m.View()
	if !strings.Contains(view, "debugging") {
		t.Errorf("expected view to contain tag 'debugging', got:\n%s", view)
	}
}
