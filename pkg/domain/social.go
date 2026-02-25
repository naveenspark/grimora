package domain

import (
	"time"

	"github.com/google/uuid"
)

// StreamEvent represents one item in the activity feed.
// Kind is one of "spell", "weapon", "member", "muse", "reject", "featured", "convo".
type StreamEvent struct {
	Kind          string    `json:"kind"`
	ID            uuid.UUID `json:"id"`
	MagicianLogin string    `json:"magician_login"`
	GuildID       string    `json:"guild_id"`
	City          string    `json:"city,omitempty"`
	Title         string    `json:"title"`
	Tag           string    `json:"tag,omitempty"`
	Upvotes       int       `json:"upvotes,omitempty"`
	Potency       int       `json:"potency,omitempty"`
	Voice         string    `json:"voice,omitempty"`         // Grimoire voice line
	Contributions int       `json:"contributions,omitempty"` // For join events
	TopLanguage   string    `json:"top_language,omitempty"`  // For join events
	CreatedAt     time.Time `json:"created_at"`
}

// MagicianCard is a magician profile enriched for the "who" browse view.
type MagicianCard struct {
	Magician
	SpellCount   int    `json:"spell_count"`
	WeaponCount  int    `json:"weapon_count"`
	IsFollowing  bool   `json:"is_following"`
	Emblem       string `json:"emblem,omitempty"`        // Guild emoji
	Move         int    `json:"move,omitempty"`          // Leaderboard rank movement
	TotalPotency int    `json:"total_potency,omitempty"` // Total potency score
	Online       bool   `json:"online,omitempty"`        // Presence status
}

// Thread is a DM conversation between two magicians.
type Thread struct {
	ID           uuid.UUID `json:"id"`
	OtherLogin   string    `json:"other_login"`
	OtherGuildID string    `json:"other_guild_id"`
	LastMessage  string    `json:"last_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// LeaderboardEntry is one row in the leaderboard ranking.
type LeaderboardEntry struct {
	Rank         int    `json:"rank"`
	Login        string `json:"login"`
	GuildID      string `json:"guild_id"`
	City         string `json:"city,omitempty"`
	DisplayName  string `json:"display_name,omitempty"`
	Archetype    string `json:"archetype"`
	SpellsForged int    `json:"spells_forged"`
	TotalPotency int    `json:"total_potency"`
	CardURL      string `json:"card_url,omitempty"`
}

// Message is a single direct message.
type Message struct {
	ID          uuid.UUID `json:"id"`
	ThreadID    uuid.UUID `json:"thread_id"`
	SenderID    uuid.UUID `json:"sender_id"`
	SenderLogin string    `json:"sender_login"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}
