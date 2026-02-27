package domain

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents a single notification event.
type Notification struct {
	ID          uuid.UUID  `json:"id"`
	RecipientID uuid.UUID  `json:"recipient_id"`
	Type        string     `json:"type"`
	ActorID     uuid.UUID  `json:"actor_id"`
	GroupKey    string     `json:"-"`
	ActorLogin  string     `json:"actor_login"`
	ActorGuild  string     `json:"actor_guild"`
	Preview     string     `json:"preview,omitempty"`
	RefID       *uuid.UUID `json:"ref_id,omitempty"`
	RefSlug     string     `json:"ref_slug,omitempty"`
	Read        bool       `json:"read"`
	CreatedAt   time.Time  `json:"created_at"`
}

// GroupedNotification extends Notification with multi-actor info for display.
type GroupedNotification struct {
	Notification
	Actors     []NotifActor `json:"actors,omitempty"`
	ActorCount int          `json:"actor_count"`
}

// NotifActor is a minimal actor representation for grouped notifications.
type NotifActor struct {
	Login   string `json:"login"`
	GuildID string `json:"guild_id"`
}

// NotificationSettings controls per-type notification delivery preferences.
type NotificationSettings struct {
	Type  string `json:"type"`
	InApp bool   `json:"in_app"`
	Push  bool   `json:"push"`
}
