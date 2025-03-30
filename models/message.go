// models/message.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID                string     `json:"id"`
	SessionID         string     `json:"session_id"`
	LocationID        string     `json:"location_id"`
	SenderUserID      string     `json:"sender_user_id,omitempty"`
	ReceiverUserID    string     `json:"receiver_user_id,omitempty"`
	SenderContactID   string     `json:"sender_contact_id,omitempty"`
	ReceiverContactID string     `json:"receiver_contact_id,omitempty"`
	Content           string     `json:"content"`
	SentAt            time.Time  `json:"sent_at"`
	ReadAt            *time.Time `json:"read_at,omitempty"`
	IsRead            bool       `json:"is_read"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
}

type DBMessage struct {
	ID                uuid.UUID  `json:"id"`
	SessionID         uuid.UUID  `json:"session_id"`
	LocationID        uuid.UUID  `json:"location_id"`
	SenderUserID      uuid.UUID  `json:"sender_user_id,omitempty"`
	ReceiverUserID    uuid.UUID  `json:"receiver_user_id,omitempty"`
	SenderContactID   uuid.UUID  `json:"sender_contact_id,omitempty"`
	ReceiverContactID uuid.UUID  `json:"receiver_contact_id,omitempty"`
	Content           string     `json:"content"`
	SentAt            time.Time  `json:"sent_at"`
	ReadAt            *time.Time `json:"read_at,omitempty"`
	IsRead            bool       `json:"is_read"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
}
