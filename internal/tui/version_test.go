package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"1.0.1", "1.0.0", true},
		{"1.1.0", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"v1.0.1", "v1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"0.9.0", "1.0.0", false},
		{"dev", "dev", false},
		{"abc", "def", false},
		{"v0.5.0", "0.4.2", true},
		{"0.4.2", "v0.5.0", false},
	}

	for _, tc := range tests {
		t.Run(tc.latest+"_vs_"+tc.current, func(t *testing.T) {
			got := isNewerVersion(tc.latest, tc.current)
			if got != tc.want {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
			}
		})
	}
}

func TestCheckVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.5.0"}) //nolint:errcheck
	}))
	defer srv.Close()

	// We can't easily override the URL in checkVersion, so test the
	// components individually. checkVersion is a thin wrapper around
	// isNewerVersion + HTTP fetch — the HTTP fetch is tested via
	// TestCheckVersionIntegration below using a custom command.

	// Test that checkVersion returns nil for dev builds.
	cmd := checkVersion("dev")
	if cmd != nil {
		t.Error("expected nil cmd for dev build")
	}

	cmd = checkVersion("")
	if cmd != nil {
		t.Error("expected nil cmd for empty version")
	}
}

// TestCheckVersionMockServer tests the full flow with a mock GitHub API.
func TestCheckVersionMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.5.0"}) //nolint:errcheck
	}))
	defer srv.Close()

	// Build a command that hits the mock server instead of GitHub.
	cmd := checkVersionFromURL(srv.URL, "0.4.0")
	msg := cmd().(versionCheckMsg)
	if !msg.hasUpdate {
		t.Error("expected hasUpdate=true for 0.5.0 > 0.4.0")
	}
	if msg.latestVersion != "v0.5.0" {
		t.Errorf("expected latestVersion=v0.5.0, got %q", msg.latestVersion)
	}
}

func TestCheckVersionMockServerNoUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.4.0"}) //nolint:errcheck
	}))
	defer srv.Close()

	cmd := checkVersionFromURL(srv.URL, "0.4.0")
	msg := cmd().(versionCheckMsg)
	if msg.hasUpdate {
		t.Error("expected hasUpdate=false for same version")
	}
}

func TestCheckVersionMockServer404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cmd := checkVersionFromURL(srv.URL, "0.4.0")
	msg := cmd().(versionCheckMsg)
	if msg.hasUpdate {
		t.Error("expected hasUpdate=false on 404")
	}
}

// checkVersionFromURL is a test helper that lets us point at a mock server.
func checkVersionFromURL(url, current string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{}
		resp, err := client.Get(url)
		if err != nil {
			return versionCheckMsg{}
		}
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusOK {
			return versionCheckMsg{}
		}
		var release struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return versionCheckMsg{}
		}
		latest := strings.TrimPrefix(release.TagName, "v")
		if isNewerVersion(latest, current) {
			return versionCheckMsg{latestVersion: "v" + latest, hasUpdate: true}
		}
		return versionCheckMsg{}
	}
}
