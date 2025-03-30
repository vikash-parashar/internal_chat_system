package models

import (
	"time"

	"github.com/google/uuid"
)

type ChatSession struct {
	ID            uuid.UUID  `json:"id"`
	ContactID     uuid.UUID  `json:"contact_id"`
	UserID        uuid.UUID  `json:"user_id"`
	LocationID    uuid.UUID  `json:"location_id"`
	StartedAt     time.Time  `json:"started_at"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
}

type InternalChatSession struct {
	ID            string     `json:"id"`
	ContactID     string     `json:"contact_id"`
	UserID        string     `json:"user_id"`
	LocationID    string     `json:"location_id"`
	StartedAt     time.Time  `json:"started_at"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
}

type ChatSessionResponse struct {
	ID            uuid.UUID  `json:"id"`
	ContactID     uuid.UUID  `json:"contact_id"`
	ContactName   string     `json:"contact_name,omitempty"`
	UserID        uuid.UUID  `json:"user_id"`
	UserName      string     `json:"user_name,omitempty"`
	LocationID    uuid.UUID  `json:"location_id"`
	StartedAt     time.Time  `json:"started_at"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	LastMessage   string     `json:"last_message,omitempty"`
	UnreadCount   int        `json:"unread_count,omitempty"`
}
