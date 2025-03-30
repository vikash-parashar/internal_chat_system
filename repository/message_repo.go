// repository/message_repo.go
package repository

import (
	"database/sql"
	"time"

	"internal_chat_system/models"
)

type MessageRepo struct {
	DB *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{DB: db}
}

func (r *MessageRepo) SaveMessage(msg *models.Message) error {
	query := `
		INSERT INTO messages (
			id, location_id, sender_user_id, receiver_user_id,
			sender_contact_id, receiver_contact_id, content, sent_at, is_read
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	msg.SentAt = time.Now()
	msg.IsRead = false

	_, err := r.DB.Exec(query,
		msg.ID, msg.LocationID, msg.SenderUserID, msg.ReceiverUserID,
		msg.SenderContactID, msg.ReceiverContactID, msg.Content,
		msg.SentAt, msg.IsRead,
	)
	return err
}

func (r *MessageRepo) GetConversation(locationID, contactID, userID string) ([]models.Message, error) {
	query := `
		SELECT id, location_id, sender_user_id, receiver_user_id,
		       sender_contact_id, receiver_contact_id, content, sent_at, read_at, is_read
		FROM messages
		WHERE location_id = $1 AND (
			(sender_user_id = $2 AND receiver_contact_id = $3) OR
			(sender_contact_id = $3 AND receiver_user_id = $2)
		)
		ORDER BY sent_at ASC
	`

	rows, err := r.DB.Query(query, locationID, userID, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(&msg.ID, &msg.LocationID, &msg.SenderUserID, &msg.ReceiverUserID,
			&msg.SenderContactID, &msg.ReceiverContactID, &msg.Content, &msg.SentAt,
			&msg.ReadAt, &msg.IsRead)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}
