package domain

import (
	"time"

	"github.com/google/uuid"
)

// PushSubscription represents a browser push notification subscription.
type PushSubscription struct {
	ID         uuid.UUID `json:"id"`
	MagicianID uuid.UUID `json:"magician_id"`
	Endpoint   string    `json:"endpoint"`
	P256dh     string    `json:"p256dh"`
	Auth       string    `json:"auth"`
	CreatedAt  time.Time `json:"created_at"`
}
