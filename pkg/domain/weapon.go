package domain

import (
	"time"

	"github.com/google/uuid"
)

// Weapon represents a tool/repository shared on Grimora.
type Weapon struct {
	ID             uuid.UUID `json:"id"`
	MagicianID     uuid.UUID `json:"magician_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	RepositoryURL  string    `json:"repository_url"`
	GitHubStars    int       `json:"github_stars"`
	GitHubForks    int       `json:"github_forks"`
	GitHubLanguage string    `json:"github_language,omitempty"`
	License        string    `json:"license,omitempty"`
	SaveCount      int       `json:"save_count"`
	CreatedAt      time.Time `json:"created_at"`
}
