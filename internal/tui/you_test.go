package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/naveenspark/grimora/pkg/domain"
)

func newTestYouModel() youModel {
	m := newYouModel(nil)
	m.width = 80
	m.height = 40
	return m
}

func makeTestProject(name, insight string) domain.WorkshopProject {
	return domain.WorkshopProject{
		ID:        uuid.New(),
		Name:      name,
		Insight:   insight,
		UpdatedAt: time.Now(),
	}
}

func TestYouIdentityShowsEmblemAndName(t *testing.T) {
	m := newTestYouModel()
	me := &domain.Magician{
		ID:          uuid.New(),
		GitHubLogin: "mywizard",
		GuildID:     "loomari",
		CardNumber:  42,
	}
	m, _ = m.Update(meLoadedMsg{me: me})

	view := m.View()
	if !strings.Contains(view, "mywizard") {
		t.Errorf("expected login 'mywizard' in view, got:\n%s", view)
	}
	// loomari has spider emblem
	if !strings.Contains(view, GuildEmblem("loomari")) {
		t.Errorf("expected guild emblem in view, got:\n%s", view)
	}
}

func TestYouWorkshopListRendersProjects(t *testing.T) {
	m := newTestYouModel()
	projects := []domain.WorkshopProject{
		makeTestProject("Distributed Cache", "Using Redis for speed"),
		makeTestProject("Auth Service", "OAuth2 implementation"),
	}
	m, _ = m.Update(workshopLoadedMsg{projects: projects})

	view := m.View()
	if !strings.Contains(view, "Distributed Cache") {
		t.Errorf("expected 'Distributed Cache' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Auth Service") {
		t.Errorf("expected 'Auth Service' in view, got:\n%s", view)
	}
}

func TestYouWorkshopEditModeEnterSaves(t *testing.T) {
	m := newTestYouModel()
	projects := []domain.WorkshopProject{makeTestProject("My Project", "Original insight")}
	m, _ = m.Update(workshopLoadedMsg{projects: projects})

	// Enter edit mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if m.wsState != wsEditing {
		t.Fatal("expected wsState=wsEditing after 'e'")
	}

	// Type new insight text
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})

	// Press Enter to save — returns a cmd (API call)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected save command on Enter, got nil")
	}
}

func TestYouWorkshopEditModeEscCancels(t *testing.T) {
	m := newTestYouModel()
	projects := []domain.WorkshopProject{makeTestProject("My Project", "Original insight")}
	m, _ = m.Update(workshopLoadedMsg{projects: projects})

	// Enter edit mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if m.wsState != wsEditing {
		t.Fatal("expected wsState=wsEditing")
	}

	// Press Esc to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.wsState != wsNormal {
		t.Errorf("expected wsState=wsNormal after Esc, got %d", m.wsState)
	}
	if m.wsAddName != "" {
		t.Errorf("expected wsAddName cleared after Esc, got %q", m.wsAddName)
	}
	if m.wsAddInsight != "" {
		t.Errorf("expected wsAddInsight cleared after Esc, got %q", m.wsAddInsight)
	}
}

func TestYouWorkshopAddModeEnterCreates(t *testing.T) {
	m := newTestYouModel()
	m, _ = m.Update(workshopLoadedMsg{projects: nil})

	// Enter add mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.wsState != wsAdding {
		t.Fatal("expected wsState=wsAdding after 'a'")
	}

	// Type project name
	for _, ch := range "MyProject" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	// Enter submits
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected create command on Enter with name, got nil")
	}
}

func TestYouWorkshopAddModeEscCancels(t *testing.T) {
	m := newTestYouModel()
	m, _ = m.Update(workshopLoadedMsg{projects: nil})

	// Enter add mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.wsState != wsAdding {
		t.Fatal("expected wsState=wsAdding")
	}

	// Press Esc to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.wsState != wsNormal {
		t.Errorf("expected wsState=wsNormal after Esc, got %d", m.wsState)
	}
}

func TestYouWorkshopDeleteModeYConfirms(t *testing.T) {
	m := newTestYouModel()
	projects := []domain.WorkshopProject{makeTestProject("DeleteMe", "")}
	m, _ = m.Update(workshopLoadedMsg{projects: projects})

	// Enter delete mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.wsState != wsDeleting {
		t.Fatal("expected wsState=wsDeleting after 'd'")
	}

	// Press 'y' to confirm — should return a command
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Error("expected delete command on 'y', got nil")
	}
}

func TestYouWorkshopDeleteModeNCancels(t *testing.T) {
	m := newTestYouModel()
	projects := []domain.WorkshopProject{makeTestProject("KeepMe", "")}
	m, _ = m.Update(workshopLoadedMsg{projects: projects})

	// Enter delete mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.wsState != wsDeleting {
		t.Fatal("expected wsState=wsDeleting")
	}

	// Press 'n' to cancel
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.wsState != wsNormal {
		t.Errorf("expected wsState=wsNormal after 'n', got %d", m.wsState)
	}
}

func TestYouDeleteLastItemAdjustsCursor(t *testing.T) {
	m := newTestYouModel()
	proj := makeTestProject("OnlyProject", "")
	m, _ = m.Update(workshopLoadedMsg{projects: []domain.WorkshopProject{proj}})

	// The deleted message adjusts cursor if it was at the last item
	m, _ = m.Update(workshopDeletedMsg{id: proj.ID.String(), err: nil})
	if m.wsCursor != 0 {
		t.Errorf("expected wsCursor=0 after delete last, got %d", m.wsCursor)
	}
	if len(m.projects) != 0 {
		t.Errorf("expected 0 projects after delete, got %d", len(m.projects))
	}
}

func TestYouViewShowsBuildJournalSection(t *testing.T) {
	m := newTestYouModel()
	projects := []domain.WorkshopProject{makeTestProject("MyProject", "testing")}
	m, _ = m.Update(workshopLoadedMsg{projects: projects})

	view := m.View()
	if !strings.Contains(view, "BUILD JOURNAL") {
		t.Errorf("expected 'BUILD JOURNAL' in view, got:\n%s", view)
	}
}

func TestYouHelpKeysChangeWithState(t *testing.T) {
	m := newTestYouModel()

	// Normal state
	normalHelp := m.helpKeys()
	if !strings.Contains(normalHelp, "j/k") {
		t.Errorf("expected 'j/k' in normal help, got %q", normalHelp)
	}

	// Editing state
	m.wsState = wsEditing
	editHelp := m.helpKeys()
	if !strings.Contains(editHelp, "save") {
		t.Errorf("expected 'save' in editing help, got %q", editHelp)
	}
	if !strings.Contains(editHelp, "cancel") {
		t.Errorf("expected 'cancel' in editing help, got %q", editHelp)
	}

	// Adding state
	m.wsState = wsAdding
	addHelp := m.helpKeys()
	if !strings.Contains(addHelp, "tab") {
		t.Errorf("expected 'tab' in adding help, got %q", addHelp)
	}

	// Deleting state
	m.wsState = wsDeleting
	deleteHelp := m.helpKeys()
	if !strings.Contains(deleteHelp, "confirm") {
		t.Errorf("expected 'confirm' in deleting help, got %q", deleteHelp)
	}
}

func TestYouInvitesSectionRenders(t *testing.T) {
	m := newTestYouModel()
	invites := []domain.Invite{
		{ID: uuid.New(), Code: "INVITE1"},
		{ID: uuid.New(), Code: "INVITE2"},
	}
	m, _ = m.Update(youInvitesLoadedMsg{invites: invites})

	view := m.View()
	if !strings.Contains(view, "INVITE") {
		t.Errorf("expected invite codes in view, got:\n%s", view)
	}
	if !strings.Contains(view, "grimora.ai/join/") {
		t.Errorf("expected invite URL in view, got:\n%s", view)
	}
}

func TestYouWorkshopDeletedUpdatesProjectList(t *testing.T) {
	m := newTestYouModel()
	p1 := makeTestProject("Project A", "insight A")
	p2 := makeTestProject("Project B", "insight B")
	m, _ = m.Update(workshopLoadedMsg{projects: []domain.WorkshopProject{p1, p2}})

	// Delete p1
	m, _ = m.Update(workshopDeletedMsg{id: p1.ID.String(), err: nil})

	if len(m.projects) != 1 {
		t.Errorf("expected 1 project after delete, got %d", len(m.projects))
	}
	if m.projects[0].Name != "Project B" {
		t.Errorf("expected remaining project 'Project B', got %q", m.projects[0].Name)
	}
}

func TestYouProjectStatusBadge(t *testing.T) {
	m := newTestYouModel()
	proj := makeTestProject("My Project", "working on it")
	m, _ = m.Update(workshopLoadedMsg{projects: []domain.WorkshopProject{proj}})

	// Simulate receiving updates with no ship
	m, _ = m.Update(projectUpdatesMsg{
		projectID: proj.ID.String(),
		updates: []domain.ProjectUpdate{
			{Kind: "update", Body: "progress"},
		},
	})

	view := m.View()
	if !strings.Contains(view, "building") {
		t.Errorf("expected 'building' badge in view, got:\n%s", view)
	}
}

func TestYouProjectStatusShipped(t *testing.T) {
	m := newTestYouModel()
	proj := makeTestProject("Shipped Thing", "done")
	m, _ = m.Update(workshopLoadedMsg{projects: []domain.WorkshopProject{proj}})

	// Simulate receiving updates with a ship
	m, _ = m.Update(projectUpdatesMsg{
		projectID: proj.ID.String(),
		updates: []domain.ProjectUpdate{
			{Kind: "update", Body: "progress"},
			{Kind: "ship", Body: "v1.0"},
		},
	})

	view := m.View()
	if !strings.Contains(view, "shipped") {
		t.Errorf("expected 'shipped' badge in view, got:\n%s", view)
	}
}

func TestYouGrimoireQuipRendered(t *testing.T) {
	m := newTestYouModel()
	me := &domain.Magician{
		ID:          uuid.New(),
		GitHubLogin: "archmage",
		GuildID:     "loomari",
		Archetype:   "architect",
	}
	m, _ = m.Update(meLoadedMsg{me: me})

	view := m.View()
	// architect quip: "you design the spells that others dare not imagine."
	if !strings.Contains(view, "design") {
		t.Errorf("expected architect quip in view, got:\n%s", view)
	}
}

func TestYouStatsBarRendered(t *testing.T) {
	m := newTestYouModel()
	m, _ = m.Update(meLoadedMsg{
		me:    &domain.Magician{ID: uuid.New(), GitHubLogin: "wizard"},
		stats: &domain.ForgeStats{SpellsForged: 12, TotalPotency: 28, Rank: 4},
	})

	view := m.View()
	if !strings.Contains(view, "CRAFT") {
		t.Errorf("expected 'CRAFT' section in view, got:\n%s", view)
	}
	if !strings.Contains(view, "P28") {
		t.Errorf("expected 'P28' potency in view, got:\n%s", view)
	}
	if !strings.Contains(view, "#4") {
		t.Errorf("expected '#4' rank in view, got:\n%s", view)
	}
}

func TestYouBuildTimelineRendered(t *testing.T) {
	m := newTestYouModel()
	proj := makeTestProject("Timeline Project", "test")
	m, _ = m.Update(workshopLoadedMsg{projects: []domain.WorkshopProject{proj}})
	m, _ = m.Update(projectUpdatesMsg{
		projectID: proj.ID.String(),
		updates: []domain.ProjectUpdate{
			{Kind: "start", Body: "", CreatedAt: time.Now().Add(-48 * time.Hour)},
			{Kind: "update", Body: "added search", CreatedAt: time.Now().Add(-24 * time.Hour)},
		},
	})

	view := m.View()
	if !strings.Contains(view, "●") {
		t.Errorf("expected timeline dot '●' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "started building") {
		t.Errorf("expected 'started building' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "added search") {
		t.Errorf("expected 'added search' in view, got:\n%s", view)
	}
}

func TestYouBuildTimelineShipMarker(t *testing.T) {
	m := newTestYouModel()
	proj := makeTestProject("Shipped Project", "done")
	m, _ = m.Update(workshopLoadedMsg{projects: []domain.WorkshopProject{proj}})
	m, _ = m.Update(projectUpdatesMsg{
		projectID: proj.ID.String(),
		updates: []domain.ProjectUpdate{
			{Kind: "start", Body: "", CreatedAt: time.Now().Add(-72 * time.Hour)},
			{Kind: "ship", Body: "v1.0", CreatedAt: time.Now().Add(-1 * time.Hour)},
		},
	})

	view := m.View()
	if !strings.Contains(view, "✦") {
		t.Errorf("expected ship marker '✦' in view, got:\n%s", view)
	}
}

func TestYouTimelineMaxUpdates(t *testing.T) {
	m := newTestYouModel()
	proj := makeTestProject("Many Updates", "busy")
	m, _ = m.Update(workshopLoadedMsg{projects: []domain.WorkshopProject{proj}})

	updates := make([]domain.ProjectUpdate, 8)
	for i := range updates {
		updates[i] = domain.ProjectUpdate{
			Kind:      "update",
			Body:      "update " + string(rune('A'+i)),
			CreatedAt: time.Now().Add(-time.Duration(8-i) * time.Hour),
		}
	}
	m, _ = m.Update(projectUpdatesMsg{
		projectID: proj.ID.String(),
		updates:   updates,
	})

	view := m.View()
	if !strings.Contains(view, "earlier") {
		t.Errorf("expected 'earlier' hint for truncated timeline, got:\n%s", view)
	}
	// Should show "3 earlier updates" (8 - 5 = 3)
	if !strings.Contains(view, "3 earlier") {
		t.Errorf("expected '3 earlier' in view, got:\n%s", view)
	}
}
