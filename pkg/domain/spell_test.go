package domain

import "testing"

func TestValidTag(t *testing.T) {
	tests := []struct {
		name  string
		tag   string
		valid bool
	}{
		{"valid debugging", "debugging", true},
		{"valid refactoring", "refactoring", true},
		{"valid architecture", "architecture", true},
		{"valid testing", "testing", true},
		{"valid devops", "devops", true},
		{"valid data", "data", true},
		{"valid frontend", "frontend", true},
		{"valid backend", "backend", true},
		{"valid security", "security", true},
		{"valid performance", "performance", true},
		{"valid general", "general", true},
		{"invalid empty", "", false},
		{"invalid unknown", "unknown", false},
		{"invalid capitalized", "Debugging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidTag(tt.tag); got != tt.valid {
				t.Errorf("ValidTag(%q) = %v, want %v", tt.tag, got, tt.valid)
			}
		})
	}
}

func TestValidTagsCount(t *testing.T) {
	if got := len(ValidTags); got != 20 {
		t.Errorf("len(ValidTags) = %d, want 20", got)
	}
}
