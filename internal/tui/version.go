package tui

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// versionCheckMsg carries the result of a background GitHub release check.
type versionCheckMsg struct {
	latestVersion string
	hasUpdate     bool
}

// checkVersion fires a non-blocking HTTP request to GitHub to see if a newer
// CLI release exists. Returns a no-op message when version is "dev".
func checkVersion(current string) tea.Cmd {
	if current == "" || current == "dev" {
		return nil
	}
	return func() tea.Msg {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("https://api.github.com/repos/naveenspark/grimora/releases/latest")
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

// isNewerVersion returns true if latest is a newer semver than current.
// Copied from cmd/grimora/main.go (can't import main package).
func isNewerVersion(latest, current string) bool {
	parse := func(v string) (int, int, int) {
		v = strings.TrimPrefix(v, "v")
		parts := strings.SplitN(v, ".", 3)
		atoi := func(s string) int {
			n, _ := strconv.Atoi(s) //nolint:errcheck
			return n
		}
		var maj, min, patch int
		if len(parts) > 0 {
			maj = atoi(parts[0])
		}
		if len(parts) > 1 {
			min = atoi(parts[1])
		}
		if len(parts) > 2 {
			patch = atoi(parts[2])
		}
		return maj, min, patch
	}
	lMaj, lMin, lPatch := parse(latest)
	cMaj, cMin, cPatch := parse(current)
	if lMaj != cMaj {
		return lMaj > cMaj
	}
	if lMin != cMin {
		return lMin > cMin
	}
	return lPatch > cPatch
}
