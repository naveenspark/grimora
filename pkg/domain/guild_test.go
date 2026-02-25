package domain

import "testing"

func TestValidGuildID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"valid loomari", "loomari", true},
		{"valid ashborne", "ashborne", true},
		{"valid amarok", "amarok", true},
		{"valid nyx", "nyx", true},
		{"valid cipher", "cipher", true},
		{"valid fathom", "fathom", true},
		{"invalid empty", "", false},
		{"invalid unknown", "unknown", false},
		{"invalid capitalized", "Loomari", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidGuildID(tt.id); got != tt.valid {
				t.Errorf("ValidGuildID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

func TestGuildsCount(t *testing.T) {
	if got := len(Guilds); got != 6 {
		t.Errorf("len(Guilds) = %d, want 6", got)
	}
}
