package domain

import "time"

// Session represents an authenticated user session.
type Session struct {
	ID         string    `json:"id"`
	MagicianID string    `json:"magician_id"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}
