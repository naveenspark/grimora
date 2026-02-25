package domain

import (
	"time"

	"github.com/google/uuid"
)

// Invite is a one-time access code that gates signup.
// created_by is nil for founder-seeded codes.
type Invite struct {
	ID        uuid.UUID  `json:"id"`
	Code      string     `json:"code"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	UsedBy    *uuid.UUID `json:"used_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// InvitesPerMagician is how many invite codes each new magician receives.
const InvitesPerMagician = 5
