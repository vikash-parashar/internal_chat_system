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
			sender_contact_id, receiver_contact_id, content, sent_at, is_read, session_id, file_url, file_name, file_type, reply_to_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	querySelectConversation = `
		SELECT id, location_id, sender_user_id, receiver_user_id,
			sender_contact_id, receiver_contact_id, content,
			sent_at, read_at, is_read,
			file_url, file_name, file_type, reply_to_id
		FROM messages
		WHERE location_id = $1 AND (
			(sender_user_id = $2 AND receiver_contact_id = $3) OR
			(sender_contact_id = $3 AND receiver_user_id = $2)
		)
		AND deleted_at IS NULL
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
		AND deleted_at IS NULL
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

	queryDeleteMessage = `UPDATE messages SET deleted_at = now() WHERE id = $1`

	queryGetReactions = `
		SELECT id, message_id, user_id, emoji, created_at
		FROM message_reactions
		WHERE message_id = $1
	`
	queryRemoveReaction = `
		DELETE FROM message_reactions
		WHERE message_id = $1 AND user_id = $2 AND emoji = $3
	`
	queryAddReaction = `
		INSERT INTO message_reactions (message_id, user_id, emoji)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id, user_id, emoji) DO NOTHING
	`
	querySelectMessageByID = `
		SELECT id, location_id, sender_user_id, receiver_user_id,
			sender_contact_id, receiver_contact_id, content,
			sent_at, read_at, is_read, file_url, file_name, file_type
		FROM messages WHERE id = $1 AND deleted_at IS NULL
	`

	queryUpdateMessageContent = `
	UPDATE messages SET content = $1, edited_at = now()
	WHERE id = $2 AND sender_user_id = $3 AND deleted_at IS NULL
	`

	queryTogglePinMessage = `
	UPDATE messages SET is_pinned = $1
	WHERE id = $2
	`
	queryGetPinnedMessages = `
		SELECT id, content, file_url, file_name, file_type, sent_at
		FROM messages
		WHERE session_id = $1 AND is_pinned = true
		ORDER BY sent_at DESC
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

	var receiverUserID, senderContactID, replyToID *uuid.UUID
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
	if msg.ReplyToID != nil {
		val, err := uuid.Parse(*msg.ReplyToID)
		if err != nil {
			log.Println("❌ Invalid UUID for ReplyToID:", err)
			return err
		}
		replyToID = &val
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
		msg.SentAt, msg.IsRead, sessionID, msg.FileURL, msg.FileName, msg.FileType, replyToID,
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
	messageMap := make(map[uuid.UUID]*models.DBMessage)

	for rows.Next() {
		var msg models.DBMessage
		var replyToID *uuid.UUID

		err := rows.Scan(&msg.ID, &msg.LocationID, &msg.SenderUserID, &msg.ReceiverUserID,
			&msg.SenderContactID, &msg.ReceiverContactID, &msg.Content,
			&msg.SentAt, &msg.ReadAt, &msg.IsRead,
			&msg.FileURL, &msg.FileName, &msg.FileType,
			&replyToID, &msg.EditedAt, &msg.IsPinned,
		)

		if err != nil {
			log.Println("❌ Failed to scan message:", err)
			return nil, err
		}

		msg.MessageType = "text"
		if msg.FileURL != "" {
			msg.MessageType = "file"
		}

		msg.Reactions, _ = r.GetReactions(msg.ID.String())
		if replyToID != nil {
			msg.ReplyToID = replyToID
		}

		messageMap[msg.ID] = &msg
		messages = append(messages, msg)
	}

	// Populate reply_to inline
	for i := range messages {
		if messages[i].ReplyToID != nil {
			if parent, ok := messageMap[*messages[i].ReplyToID]; ok {
				msgCopy := *parent
				messages[i].ReplyTo = &msgCopy
			}
		}
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

func (r *MessageRepo) DeleteMessage(id uuid.UUID) error {

	_, err := r.DB.Exec(queryDeleteMessage, id)
	return err
}

func (r *MessageRepo) AddReaction(msgID, userID, emoji string) error {
	log.Printf("➕ Adding reaction: %s by user %s to message %s", emoji, userID, msgID)

	_, err := r.DB.Exec(queryAddReaction, msgID, userID, emoji)
	if err != nil {
		log.Printf("❌ Failed to add reaction: %v", err)
	} else {
		log.Printf("✅ Reaction added: %s by user %s", emoji, userID)
	}
	return err
}

func (r *MessageRepo) RemoveReaction(msgID, userID, emoji string) error {
	log.Printf("❌ Removing reaction: %s by user %s from message %s", emoji, userID, msgID)

	_, err := r.DB.Exec(queryRemoveReaction, msgID, userID, emoji)
	if err != nil {
		log.Printf("❌ Failed to remove reaction: %v", err)
	} else {
		log.Printf("✅ Reaction removed: %s by user %s", emoji, userID)
	}
	return err
}

func (r *MessageRepo) GetReactions(msgID string) ([]models.MessageReaction, error) {
	log.Printf("🔍 Fetching reactions for message %s", msgID)

	rows, err := r.DB.Query(queryGetReactions, msgID)
	if err != nil {
		log.Printf("❌ Failed to fetch reactions: %v", err)
		return nil, err
	}
	defer rows.Close()

	var reactions []models.MessageReaction
	for rows.Next() {
		var r models.MessageReaction
		if err := rows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Emoji, &r.CreatedAt); err != nil {
			log.Printf("❌ Failed to scan reaction row: %v", err)
			return nil, err
		}
		reactions = append(reactions, r)
	}

	log.Printf("✅ Found %d reaction(s) for message %s", len(reactions), msgID)
	return reactions, nil
}

func (r *MessageRepo) GetMessageByID(id string) (models.DBMessage, error) {
	var msg models.DBMessage
	err := r.DB.QueryRow(querySelectMessageByID, id).Scan(
		&msg.ID, &msg.LocationID, &msg.SenderUserID, &msg.ReceiverUserID,
		&msg.SenderContactID, &msg.ReceiverContactID, &msg.Content,
		&msg.SentAt, &msg.ReadAt, &msg.IsRead,
		&msg.FileURL, &msg.FileName, &msg.FileType,
	)
	if err != nil {
		log.Println("❌ Failed to fetch reply message:", err)
		return msg, err
	}
	msg.MessageType = "text"
	if msg.FileURL != "" {
		msg.MessageType = "file"
	}
	return msg, nil
}

func (r *MessageRepo) UpdateMessageContent(msgID, senderID, newContent string) error {
	log.Printf("✏️ Editing message %s by user %s", msgID, senderID)
	_, err := r.DB.Exec(queryUpdateMessageContent, newContent, msgID, senderID)
	if err != nil {
		log.Printf("❌ Failed to edit message: %v", err)
	}
	return err
}

func (r *MessageRepo) TogglePinMessage(msgID string, pin bool) error {
	log.Printf("📌 Pin status update for message %s to %v", msgID, pin)
	_, err := r.DB.Exec(queryTogglePinMessage, pin, msgID)
	if err != nil {
		log.Printf("❌ Failed to update pin status: %v", err)
	}
	return err
}

func (r *MessageRepo) GetPinnedMessages(sessionID string) ([]models.PinnedMessage, error) {
	log.Printf("📍 Fetching pinned messages for session %s", sessionID)
	rows, err := r.DB.Query(queryGetPinnedMessages, sessionID)
	if err != nil {
		log.Printf("❌ Failed to fetch pinned messages: %v", err)
		return nil, err
	}
	defer rows.Close()

	var pinned []models.PinnedMessage
	for rows.Next() {
		var m models.PinnedMessage
		err := rows.Scan(&m.ID, &m.Content, &m.FileURL, &m.FileName, &m.FileType, &m.SentAt)
		if err != nil {
			log.Printf("❌ Failed to scan pinned message: %v", err)
			return nil, err
		}
		pinned = append(pinned, m)
	}

	return pinned, nil
}

func (r *MessageRepo) GetDeviceToken(userID string) (string, error) {
	var token string
	err := r.DB.QueryRow("SELECT token FROM device_tokens WHERE user_id = $1", userID).Scan(&token)
	if err != nil {
		log.Printf("❌ GetDeviceToken error: %v", err)
	}
	return token, err
}

func (r *MessageRepo) UpsertDeviceToken(userID, token string) error {
	query := `
		INSERT INTO device_tokens (user_id, token, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id) DO UPDATE
		SET token = EXCLUDED.token,
		    updated_at = now()
	`
	_, err := r.DB.Exec(query, userID, token)
	if err != nil {
		log.Printf("❌ Failed to upsert device token: %v", err)
	}
	return err
}
