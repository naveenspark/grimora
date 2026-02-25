package domain

import (
	"time"

	"github.com/google/uuid"
)

// WorkshopProject is a personal project tracked on a magician's profile.
type WorkshopProject struct {
	ID         uuid.UUID `json:"id"`
	MagicianID uuid.UUID `json:"magician_id"`
	Name       string    `json:"name"`
	Insight    string    `json:"insight"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
