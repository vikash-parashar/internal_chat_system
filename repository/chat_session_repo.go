package repository

import (
	"database/sql"
	"internal_chat_system/models"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ChatSessionRepo struct {
	DB *sql.DB
}

func NewChatSessionRepo(db *sql.DB) *ChatSessionRepo {
	return &ChatSessionRepo{DB: db}
}

const (
	queryGetSession = `
		SELECT id FROM chat_sessions
		WHERE contact_id = $1 AND user_id = $2 AND location_id = $3
	`

	queryInsertSession = `
		INSERT INTO chat_sessions (id, contact_id, user_id, location_id, started_at, last_message_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	queryUpdateLastMessage = `
		UPDATE chat_sessions SET last_message_at = $2 WHERE id = $1
	`

	queryMarkMessagesDelivered = `UPDATE messages SET delivered_at = now() WHERE id = ANY($1)`

	queryAdminListAllSessions = `
		SELECT cs.id, cs.contact_id, COALESCE(c.full_name, '') AS contact_name,
		       cs.user_id, COALESCE(u.full_name, '') AS user_name,
		       cs.location_id, cs.started_at, cs.last_message_at,
		       COALESCE(m.content, '') AS last_message,
		       (
				   SELECT COUNT(*) FROM messages
				   WHERE session_id = cs.id AND is_read = false
		       ) AS unread_count
		FROM chat_sessions cs
		LEFT JOIN contacts c ON cs.contact_id = c.id
		LEFT JOIN users u ON cs.user_id = u.id
		LEFT JOIN LATERAL (
			SELECT content FROM messages
			WHERE session_id = cs.id
			ORDER BY sent_at DESC LIMIT 1
		) m ON true
		WHERE ($1 = '' OR cs.location_id = $1)
		ORDER BY cs.last_message_at DESC NULLS LAST, cs.started_at DESC
		LIMIT $2 OFFSET $3
	`
)

func (r *ChatSessionRepo) GetOrCreateSession(contactID, userID, locationID string) (string, error) {
	var sessionID string
	err := r.DB.QueryRow(queryGetSession, contactID, userID, locationID).Scan(&sessionID)
	if err == sql.ErrNoRows {
		log.Println("📁 No session found — creating new one")
		id := uuid.New().String()
		t := time.Now()
		err := r.DB.QueryRow(queryInsertSession, id, contactID, userID, locationID, t, t).Scan(&sessionID)
		if err != nil {
			log.Println("❌ Failed to create new session:", err)
			return "", err
		}
		return sessionID, nil
	} else if err != nil {
		log.Println("❌ Failed to fetch session:", err)
		return "", err
	}

	// Session exists: update last_message_at
	_, err = r.DB.Exec(queryUpdateLastMessage, sessionID, time.Now())
	if err != nil {
		log.Println("⚠️ Failed to update last_message_at:", err)
	}

	return sessionID, nil
}

func (r *MessageRepo) MarkMessagesDelivered(ids []uuid.UUID) error {

	_, err := r.DB.Exec(queryMarkMessagesDelivered, pq.Array(ids))
	return err
}

func (r *MessageRepo) AdminListAllSessions(locationID string, limit, offset int) ([]models.ChatSessionResponse, error) {

	rows, err := r.DB.Query(queryAdminListAllSessions, locationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.ChatSessionResponse
	for rows.Next() {
		var s models.ChatSessionResponse
		if err := rows.Scan(&s.ID, &s.ContactID, &s.ContactName, &s.UserID, &s.UserName,
			&s.LocationID, &s.StartedAt, &s.LastMessageAt, &s.LastMessage, &s.UnreadCount); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}
