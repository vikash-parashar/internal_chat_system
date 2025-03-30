// models/message.go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID                string            `json:"id"`
	LocationID        string            `json:"location_id"`
	SenderUserID      string            `json:"sender_user_id"`
	ReceiverUserID    string            `json:"receiver_user_id,omitempty"`
	SenderContactID   string            `json:"sender_contact_id,omitempty"`
	ReceiverContactID string            `json:"receiver_contact_id,omitempty"`
	Content           string            `json:"content"`
	SessionID         string            `json:"session_id,omitempty"`
	SentAt            time.Time         `json:"sent_at"`
	ReadAt            *time.Time        `json:"read_at,omitempty"`
	DeliveredAt       *time.Time        `json:"delivered_at,omitempty"`
	IsRead            bool              `json:"is_read"`
	FileURL           string            `json:"file_url,omitempty"`
	FileName          string            `json:"file_name,omitempty"`
	FileType          string            `json:"file_type,omitempty"`
	ReplyToID         *string           `json:"reply_to_id,omitempty"`
	EditedAt          *time.Time        `json:"edited_at,omitempty"`
	IsPinned          bool              `json:"is_pinned"`
	MessageType       string            `json:"message_type"`
	Reactions         []MessageReaction `json:"reactions"`
	ReplyTo           *DBMessage        `json:"reply_to,omitempty"`
}

type DBMessage struct {
	ID                uuid.UUID         `json:"id"`
	LocationID        uuid.UUID         `json:"location_id"`
	SenderUserID      uuid.UUID         `json:"sender_user_id"`
	ReceiverUserID    uuid.UUID         `json:"receiver_user_id,omitempty"`
	SenderContactID   uuid.UUID         `json:"sender_contact_id,omitempty"`
	ReceiverContactID uuid.UUID         `json:"receiver_contact_id,omitempty"`
	SessionID         uuid.UUID         `json:"session_id,omitempty"`
	Content           string            `json:"content"`
	SentAt            time.Time         `json:"sent_at"`
	ReadAt            *time.Time        `json:"read_at,omitempty"`
	DeliveredAt       *time.Time        `json:"delivered_at,omitempty"`
	IsRead            bool              `json:"is_read"`
	FileURL           string            `json:"file_url,omitempty"`
	FileName          string            `json:"file_name,omitempty"`
	FileType          string            `json:"file_type,omitempty"`
	ReplyToID         *uuid.UUID        `json:"reply_to_id,omitempty"`
	EditedAt          *time.Time        `json:"edited_at,omitempty"`
	IsPinned          bool              `json:"is_pinned"`
	MessageType       string            `json:"message_type"`
	Reactions         []MessageReaction `json:"reactions"`
	ReplyTo           *DBMessage        `json:"reply_to,omitempty"`
}

type PinnedMessage struct {
	ID       string    `json:"id"`
	Content  string    `json:"content"`
	FileURL  string    `json:"file_url,omitempty"`
	FileName string    `json:"file_name,omitempty"`
	FileType string    `json:"file_type,omitempty"`
	SentAt   time.Time `json:"sent_at"`
}
