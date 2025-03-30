package repository

import (
	"database/sql"
	"fmt"
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
			sender_contact_id, receiver_contact_id, content, sent_at, is_read, session_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	querySelectConversation = `
		SELECT id, location_id, sender_user_id, receiver_user_id,
			sender_contact_id, receiver_contact_id, content,
			sent_at, read_at, delivered_at, is_read
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

	baseSessionQuery = `
		SELECT
			cs.id,
			cs.contact_id,
			COALESCE(c.full_name, '') AS contact_name,
			cs.user_id,
			COALESCE(u.full_name, '') AS user_name,
			cs.location_id,
			cs.started_at,
			cs.last_message_at,
			COALESCE(m.content, '') AS last_message,
			(
				SELECT COUNT(*) FROM messages
				WHERE session_id = cs.id AND is_read = false AND receiver_user_id = $2
			) AS unread_count
		FROM chat_sessions cs
		LEFT JOIN contacts c ON cs.contact_id = c.id
		LEFT JOIN users u ON cs.user_id = u.id
		LEFT JOIN LATERAL (
			SELECT content FROM messages
			WHERE session_id = cs.id
			ORDER BY sent_at DESC LIMIT 1
		) m ON true
		WHERE (cs.contact_id = $1 OR cs.user_id = $2)
		AND ($3 = '' OR cs.location_id = $3)
		ORDER BY cs.last_message_at DESC NULLS LAST, cs.started_at DESC
	`

	querySearchMessage = `
		SELECT id, location_id, sender_user_id, receiver_user_id,
		       sender_contact_id, receiver_contact_id, content,
		       sent_at, read_at, delivered_at, is_read
		FROM messages
		WHERE content ILIKE '%' || $1 || '%'
		AND location_id = $2
		AND (
			(sender_user_id = $3 AND receiver_contact_id = $4) OR
			(sender_contact_id = $4 AND receiver_user_id = $3)
		)
		ORDER BY sent_at DESC
		LIMIT $5
	`
)

type MessageRepo struct {
	DB *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{DB: db}
}

func (r *MessageRepo) SaveMessage(msg *models.Message) error {
	log.Printf("💾 Saving message from user %s to contact %s (session: %s)", msg.SenderUserID, msg.ReceiverContactID, msg.SessionID)

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

	sessionID, err := uuid.Parse(msg.SessionID)
	if err != nil {
		log.Println("❌ Invalid UUID for SessionID:", err)
		return err
	}

	msg.SentAt = time.Now()
	msg.IsRead = false

	_, err = r.DB.Exec(queryInsertMessage,
		id, locationID, senderUserID, receiverUserID,
		senderContactID, receiverContactID, msg.Content,
		msg.SentAt, msg.IsRead, sessionID,
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
			&msg.ReadAt, &msg.DeliveredAt, &msg.IsRead)

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

func (r *MessageRepo) ListEnrichedChatSessionsWithFilter(userID, contactID, locationID string, limit, offset int) ([]models.ChatSessionResponse, error) {
	log.Printf("🔍 Listing sessions for user=%s contact=%s location=%s limit=%d offset=%d", userID, contactID, locationID, limit, offset)

	query := fmt.Sprintf("%s LIMIT $4 OFFSET $5", baseSessionQuery)
	rows, err := r.DB.Query(query, contactID, userID, locationID, limit, offset)
	if err != nil {
		log.Println("❌ Query failed for ListEnrichedChatSessionsWithFilter:", err)
		return nil, err
	}
	defer rows.Close()

	sessions := []models.ChatSessionResponse{}
	for rows.Next() {
		var s models.ChatSessionResponse
		err := rows.Scan(&s.ID, &s.ContactID, &s.ContactName, &s.UserID, &s.UserName, &s.LocationID, &s.StartedAt, &s.LastMessageAt, &s.LastMessage, &s.UnreadCount)
		if err != nil {
			log.Println("❌ Failed to scan chat session row:", err)
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (r *MessageRepo) SearchMessages(userID, contactID, locationID, query string, limit int) ([]models.DBMessage, error) {

	rows, err := r.DB.Query(querySearchMessage, query, locationID, userID, contactID, limit)
	if err != nil {
		log.Println("❌ Search query failed:", err)
		return nil, err
	}
	defer rows.Close()

	var results []models.DBMessage
	for rows.Next() {
		var msg models.DBMessage
		err := rows.Scan(&msg.ID, &msg.LocationID, &msg.SenderUserID, &msg.ReceiverUserID,
			&msg.SenderContactID, &msg.ReceiverContactID, &msg.Content, &msg.SentAt,
			&msg.ReadAt, &msg.DeliveredAt, &msg.IsRead)
		if err != nil {
			log.Println("❌ Error scanning search row:", err)
			return nil, err
		}
		results = append(results, msg)
	}
	return results, nil
}
