package domain

import (
	"time"

	"github.com/google/uuid"
)

// Magician represents a registered Grimora user.
type Magician struct {
	ID          uuid.UUID  `json:"id"`
	GitHubID    int64      `json:"github_id"`
	GitHubLogin string     `json:"github_login"`
	CardNumber  int        `json:"card_number"`
	GuildID     string     `json:"guild_id"`
	Archetype   string     `json:"archetype"`
	Email       string     `json:"email,omitempty"`
	City        string     `json:"city,omitempty"`
	DisplayName string     `json:"display_name,omitempty"`
	Stack       []string   `json:"stack,omitempty"`
	Edition     string     `json:"edition,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
	// Ceremony fields (migration 005)
	CardURL     string   `json:"card_url,omitempty"`
	CardStatus  string   `json:"card_status,omitempty"`
	TopLanguage string   `json:"top_language,omitempty"`
	Fragments   []string `json:"fragments,omitempty"`
}
