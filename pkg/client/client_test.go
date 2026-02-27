package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/naveenspark/grimora/pkg/domain"
)

func TestGetMe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/me" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"}) //nolint:errcheck
			return
		}
		json.NewEncoder(w).Encode(domain.Magician{ //nolint:errcheck
			GitHubLogin: "testmage",
			GuildID:     "cipher",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	me, err := c.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe() error: %v", err)
	}
	if me.GitHubLogin != "testmage" {
		t.Errorf("GitHubLogin = %q, want %q", me.GitHubLogin, "testmage")
	}
	if me.GuildID != "cipher" {
		t.Errorf("GuildID = %q, want %q", me.GuildID, "cipher")
	}
}

func TestGetMe_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"}) //nolint:errcheck
	}))
	defer srv.Close()

	c := New(srv.URL, "bad-token")
	_, err := c.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
	if got := err.Error(); !strings.Contains(got, "HTTP 401") {
		t.Errorf("error = %q, want it to contain 'HTTP 401'", got)
	}
}

func TestListSpells(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/spells" {
			http.NotFound(w, r)
			return
		}
		spells := []domain.Spell{
			{Text: "debug this", Tag: "debugging"},
			{Text: "test that", Tag: "testing"},
		}
		json.NewEncoder(w).Encode(spells) //nolint:errcheck
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	spells, err := c.ListSpells(context.Background(), "", "new", 50, 0)
	if err != nil {
		t.Fatalf("ListSpells() error: %v", err)
	}
	if len(spells) != 2 {
		t.Fatalf("got %d spells, want 2", len(spells))
	}
	if spells[0].Tag != "debugging" {
		t.Errorf("spells[0].Tag = %q, want %q", spells[0].Tag, "debugging")
	}
}

func TestListSpells_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]domain.Spell{}) //nolint:errcheck
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	spells, err := c.ListSpells(context.Background(), "", "", 50, 0)
	if err != nil {
		t.Fatalf("ListSpells() error: %v", err)
	}
	if len(spells) != 0 {
		t.Errorf("got %d spells, want 0", len(spells))
	}
}

func TestCreateSpell(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req CreateSpellRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(domain.Spell{ //nolint:errcheck
			Text: req.Text,
			Tag:  req.Tag,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	spell, err := c.CreateSpell(context.Background(), CreateSpellRequest{
		Text: "my spell",
		Tag:  "debugging",
	})
	if err != nil {
		t.Fatalf("CreateSpell() error: %v", err)
	}
	if spell.Text != "my spell" {
		t.Errorf("spell.Text = %q, want %q", spell.Text, "my spell")
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "boom"}) //nolint:errcheck
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	_, err := c.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if got := err.Error(); !strings.Contains(got, "boom") {
		t.Errorf("error = %q, want it to contain 'boom'", got)
	}
}

func TestDoRequest_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)                  // slow server
		json.NewEncoder(w).Encode(domain.Magician{}) //nolint:errcheck
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.GetMe(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
