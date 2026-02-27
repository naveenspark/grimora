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
	URL        string    `json:"url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ProjectUpdate is a timeline entry on a workshop project.
type ProjectUpdate struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Kind      string    `json:"kind"` // "start", "update", "ship"
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
