package domain

import (
	"time"

	"github.com/google/uuid"
)

// Comment is a comment on a spell.
type Comment struct {
	ID        uuid.UUID `json:"id"`
	SpellID   uuid.UUID `json:"spell_id"`
	Login     string    `json:"login"`
	GuildID   string    `json:"guild_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
