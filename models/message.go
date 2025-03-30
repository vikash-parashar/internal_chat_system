// models/message.go
package models

import "time"

type Message struct {
	ID                string     `json:"id"`
	LocationID        string     `json:"location_id"`
	SenderUserID      string     `json:"sender_user_id,omitempty"`
	ReceiverUserID    string     `json:"receiver_user_id,omitempty"`
	SenderContactID   string     `json:"sender_contact_id,omitempty"`
	ReceiverContactID string     `json:"receiver_contact_id,omitempty"`
	Content           string     `json:"content"`
	SentAt            time.Time  `json:"sent_at"`
	ReadAt            *time.Time `json:"read_at,omitempty"`
	IsRead            bool       `json:"is_read"`
}
