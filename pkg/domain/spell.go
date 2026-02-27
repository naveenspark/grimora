package domain

import (
	"time"

	"github.com/google/uuid"
)

// Spell represents a shared prompt.
type Spell struct {
	ID         uuid.UUID `json:"id"`
	MagicianID uuid.UUID `json:"magician_id"`
	Text       string    `json:"text"`
	Tag        string    `json:"tag"`
	Model      string    `json:"model,omitempty"`
	Stack      []string  `json:"stack,omitempty"`
	Context    string    `json:"context,omitempty"`
	Potency    int       `json:"potency"`
	Status     string    `json:"status"` // "pending", "published", "removed"
	Upvotes    int       `json:"upvotes"`
	Preview    string    `json:"preview,omitempty"`    // Truncated text for list views
	Voice      string    `json:"voice,omitempty"`      // Grimoire commentary
	Situations string    `json:"situations,omitempty"` // LLM-generated search situations
	Author     *Author   `json:"author,omitempty"`     // Author info for display
	Comments   []Comment `json:"comments,omitempty"`   // Spell comments
	CreatedAt  time.Time `json:"created_at"`
}

// Valid spell tags.
var ValidTags = []string{
	// Code
	"debugging",
	"refactoring",
	"architecture",
	"testing",
	"devops",
	"data",
	"frontend",
	"backend",
	"security",
	"performance",
	// Creative
	"image-gen",
	"writing",
	// Strategy
	"business",
	"productivity",
	"analysis",
	// Meta
	"system-prompt",
	"education",
	// Other
	"coding",
	"conversation",
	// Catch-all
	"general",
}

// TagStat holds per-tag counts.
type TagStat struct {
	Tag          string `json:"tag"`
	SpellCount   int    `json:"spell_count"`
	TotalUpvotes int    `json:"total_upvotes"`
}

var validTagSet = func() map[string]bool {
	m := make(map[string]bool, len(ValidTags))
	for _, t := range ValidTags {
		m[t] = true
	}
	return m
}()

// ValidTag returns true if the given tag is a known spell tag.
func ValidTag(tag string) bool {
	return validTagSet[tag]
}

// SpellMatch is a spell candidate returned by semantic similarity search.
type SpellMatch struct {
	ID         uuid.UUID `json:"id"`
	Tag        string    `json:"tag"`
	Context    string    `json:"context"`
	Upvotes    int       `json:"upvotes"`
	Similarity float64   `json:"similarity"`
	CreatedAt  time.Time `json:"created_at"`
	Author     Author    `json:"author"`
}

// CastResult is the single best spell returned by the /api/cast endpoint.
// It includes the full spell text (unlike SpellMatch which omits it).
type CastResult struct {
	ID         uuid.UUID `json:"id"`
	Tag        string    `json:"tag"`
	Context    string    `json:"context"`
	Text       string    `json:"text"`
	Upvotes    int       `json:"upvotes"`
	Similarity float64   `json:"similarity"`
	CreatedAt  time.Time `json:"created_at"`
	Author     Author    `json:"author"`
}

// ForgeVerdict is the Grimoire's judgment on a submitted spell.
type ForgeVerdict struct {
	Verdict     string      `json:"verdict"`               // "ACCEPT" or "REJECT"
	Potency     int         `json:"potency,omitempty"`     // 1-3 (0 if rejected)
	Inscription string      `json:"inscription,omitempty"` // Grimoire's one-liner
	Reason      string      `json:"reason,omitempty"`      // Why rejected
	Spell       *Spell      `json:"spell,omitempty"`       // The created spell (only if accepted)
	Stats       *ForgeStats `json:"stats,omitempty"`       // Magician's forge stats
}

// ForgeStats tracks a magician's forging record and competitive rank.
type ForgeStats struct {
	SpellsForged   int     `json:"spells_forged"`
	TotalPotency   int     `json:"total_potency"`
	AvgPotency     float64 `json:"avg_potency"`
	AcceptanceRate float64 `json:"acceptance_rate"`
	Rank           int     `json:"rank"`
	TotalRanked    int     `json:"total_ranked"`
}

// Author is the magician who created a spell.
type Author struct {
	Login        string `json:"login"`
	GuildID      string `json:"guild_id"`
	City         string `json:"city,omitempty"`
	Archetype    string `json:"archetype"`
	DisplayName  string `json:"display_name,omitempty"`
	SpellsForged int    `json:"spells_forged"`
	TotalPotency int    `json:"total_potency"`
	Rank         int    `json:"rank"`
}
