package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Room is a chat room — hall (global), guild (per-guild), or topic (user-created).
type Room struct {
	ID              uuid.UUID  `json:"id"`
	Slug            string     `json:"slug"`
	Name            string     `json:"name"`
	RoomType        string     `json:"room_type"`          // "hall", "guild", "topic"
	GuildID         string     `json:"guild_id,omitempty"` // set for guild rooms
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	Description     string     `json:"description,omitempty"`
	SlowmodeSeconds int        `json:"slowmode_seconds,omitempty"`
	MaxMembers      int        `json:"max_members,omitempty"` // 0 = unlimited
	Archived        bool       `json:"archived,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// RoomMessage is a single message in a room.
type RoomMessage struct {
	ID          uuid.UUID       `json:"id"`
	RoomID      uuid.UUID       `json:"room_id"`
	SenderID    uuid.UUID       `json:"sender_id"`
	SenderLogin string          `json:"sender_login"`
	SenderGuild string          `json:"sender_guild,omitempty"`
	Body        string          `json:"body"`
	Kind        string          `json:"kind"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Reaction represents a mash reaction on a room message.
type Reaction struct {
	ID         uuid.UUID `json:"id"`
	MessageID  uuid.UUID `json:"message_id"`
	MagicianID uuid.UUID `json:"magician_id"`
	Emoji      string    `json:"emoji"`
	CreatedAt  time.Time `json:"created_at"`
}

// RoomMember is an explicit member of a topic room.
type RoomMember struct {
	RoomID     uuid.UUID `json:"room_id"`
	MagicianID uuid.UUID `json:"magician_id"`
	Role       string    `json:"role"` // "member", "moderator", "creator"
	JoinedAt   time.Time `json:"joined_at"`
}
