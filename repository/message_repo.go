// repository/message_repo.go
package repository

import (
	"database/sql"
	"log"
	"time"

	"internal_chat_system/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	queryInsertMessage = `
		INSERT INTO messages (
			id, location_id, sender_user_id, receiver_user_id,
			sender_contact_id, receiver_contact_id, content, sent_at, is_read
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	querySelectConversation = `
		SELECT id, location_id, sender_user_id, receiver_user_id,
			sender_contact_id, receiver_contact_id, content, sent_at, read_at, is_read
		FROM messages
		WHERE location_id = $1 AND (
			(sender_user_id = $2 AND receiver_contact_id = $3) OR
			(sender_contact_id = $3 AND receiver_user_id = $2)
		)
		ORDER BY sent_at ASC
	`

	queryUpdateMarkMessagesRead = `
		UPDATE messages SET is_read = true, read_at = now()
		WHERE id = ANY($1)
	`
)

type MessageRepo struct {
	DB *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{DB: db}
}

func (r *MessageRepo) SaveMessage(msg *models.Message) error {
	log.Println("📥 Saving message to DB")

	id, err := uuid.Parse(msg.ID)
	if err != nil {
		log.Println("❌ Invalid UUID for ID:", err)
		return err
	}
	locationID, err := uuid.Parse(msg.LocationID)
	if err != nil {
		log.Println("❌ Invalid UUID for LocationID:", err)
		return err
	}
	senderUserID, err := uuid.Parse(msg.SenderUserID)
	if err != nil {
		log.Println("❌ Invalid UUID for SenderUserID:", err)
		return err
	}
	receiverContactID, err := uuid.Parse(msg.ReceiverContactID)
	if err != nil {
		log.Println("❌ Invalid UUID for ReceiverContactID:", err)
		return err
	}

	var receiverUserID, senderContactID *uuid.UUID
	if msg.ReceiverUserID != "" {
		val, err := uuid.Parse(msg.ReceiverUserID)
		if err != nil {
			log.Println("❌ Invalid UUID for ReceiverUserID:", err)
			return err
		}
		receiverUserID = &val
	}
	if msg.SenderContactID != "" {
		val, err := uuid.Parse(msg.SenderContactID)
		if err != nil {
			log.Println("❌ Invalid UUID for SenderContactID:", err)
			return err
		}
		senderContactID = &val
	}

	msg.SentAt = time.Now()
	msg.IsRead = false

	_, err = r.DB.Exec(queryInsertMessage,
		id, locationID, senderUserID, receiverUserID,
		senderContactID, receiverContactID, msg.Content,
		msg.SentAt, msg.IsRead,
	)
	if err != nil {
		log.Println("❌ Failed to insert message:", err)
	}
	return err
}

func (r *MessageRepo) GetConversation(locationID, contactID, userID string) ([]models.DBMessage, error) {
	log.Printf("📤 Fetching conversation for locationID=%s, contactID=%s, userID=%s", locationID, contactID, userID)

	rows, err := r.DB.Query(querySelectConversation, locationID, userID, contactID)
	if err != nil {
		log.Println("❌ Failed to fetch messages:", err)
		return nil, err
	}
	defer rows.Close()

	var messages []models.DBMessage
	for rows.Next() {
		var msg models.DBMessage
		err := rows.Scan(&msg.ID, &msg.LocationID, &msg.SenderUserID, &msg.ReceiverUserID,
			&msg.SenderContactID, &msg.ReceiverContactID, &msg.Content, &msg.SentAt,
			&msg.ReadAt, &msg.IsRead)
		if err != nil {
			log.Println("❌ Failed to scan row:", err)
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (r *MessageRepo) MarkMessagesRead(ids []string) error {
	log.Println("📌 Marking messages as read:", ids)

	var uuids []uuid.UUID
	for _, id := range ids {
		u, err := uuid.Parse(id)
		if err != nil {
			log.Println("❌ Error parsing UUID:", err)
			return err
		}
		uuids = append(uuids, u)
	}

	_, err := r.DB.Exec(queryUpdateMarkMessagesRead, pq.Array(ids))
	if err != nil {
		log.Println("❌ Failed to mark messages read:", err)
	}
	return err
}
