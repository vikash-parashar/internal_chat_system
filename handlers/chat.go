// handlers/chat.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"internal_chat_system/internal/s3"
	"internal_chat_system/middleware/auth"
	"internal_chat_system/models"
	"internal_chat_system/notifications"
	"internal_chat_system/presence"
	"internal_chat_system/redis"
	"internal_chat_system/repository"
	"internal_chat_system/ws"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var (
	messageRepo *repository.MessageRepo
	sessionRepo *repository.ChatSessionRepo
)

func Init(repo *repository.MessageRepo, session *repository.ChatSessionRepo) {
	messageRepo = repo
	sessionRepo = session
}

func SendMessage(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg models.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		msg.ID = uuid.New().String()

		auth := auth.GetAuthContext(r)
		log.Printf("🔐 Authenticated User: ID=%s, Type=%s", auth.UserID, auth.UserType)

		// Enforce access rules
		if auth.UserType == "DOCTOR" && auth.UserID != msg.SenderUserID {
			writeError(w, http.StatusForbidden, "Unauthorized doctor")
			return
		}
		if auth.UserType == "PATIENT" && auth.UserID != msg.ReceiverContactID {
			writeError(w, http.StatusForbidden, "Unauthorized patient")
			return
		}

		if msg.LocationID == "" || msg.SenderUserID == "" || msg.ReceiverContactID == "" {
			writeError(w, http.StatusBadRequest, "Missing required fields")
			return
		}

		if msg.Content == "" && msg.FileURL == "" {
			writeError(w, http.StatusBadRequest, "Either message content or file must be provided")
			return
		}

		sessionID, err := sessionRepo.GetOrCreateSession(msg.ReceiverContactID, msg.SenderUserID, msg.LocationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create or fetch session")
			return
		}
		msg.SessionID = sessionID

		if err := messageRepo.SaveMessage(&msg); err != nil {
			log.Printf("❌ DB Error on SaveMessage: %v", err)
			writeError(w, http.StatusInternalServerError, "Could not save message")
			return
		}

		data, _ := json.Marshal(msg)
		targetID := msg.ReceiverContactID
		targetType := "contact"
		if msg.ReceiverUserID != "" {
			targetID = msg.ReceiverUserID
			targetType = "user"
		}

		if redis.IsClientConnected(msg.LocationID, targetID, hub.Clients) {
			log.Printf("🚀 Delivering message live to %s:%s", targetType, targetID)
			hub.Broadcast <- ws.BroadcastMessage{
				LocationID: msg.LocationID,
				Message:    msg,
			}
		} else {
			log.Printf("📥 Queuing offline message for %s:%s", targetType, targetID)
			_ = redis.QueueOfflineMessage(targetType, msg.LocationID, targetID, data)
		}

		if !presence.IsUserOnline(targetID) {
			token, err := messageRepo.GetDeviceToken(targetID) // You must implement this
			if err == nil && token != "" {
				notifications.SendPush(token, "New message", msg.Content)
			}
		}

		_ = redis.PublishPushEvent(redis.PushEvent{
			MessageID:    msg.ID,
			LocationID:   msg.LocationID,
			ReceiverID:   targetID,
			ReceiverType: targetType,
			Content:      msg.Content,
		})

		writeJSON(w, http.StatusCreated, msg)
	}
}

// func HandleWebSocket(hub *ws.Hub) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		locationID := r.URL.Query().Get("location_id")
// 		userID := r.URL.Query().Get("user_id")
// 		contactID := r.URL.Query().Get("contact_id")

// 		conn, err := upgrader.Upgrade(w, r, nil)
// 		if err != nil {
// 			writeError(w, http.StatusInternalServerError, "WebSocket Upgrade Failed")
// 			return
// 		}

// 		client := &ws.Client{
// 			Conn:       conn,
// 			Send:       make(chan []byte),
// 			UserID:     userID,
// 			ContactID:  contactID,
// 			LocationID: locationID,
// 			Hub:        hub,
// 		}

// 		hub.Register <- client

// 		go client.ReadPump()
// 		go client.WritePump()
// 	}
// }

func HandleWebSocket(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.URL.Query().Get("location_id")
		userID := r.URL.Query().Get("user_id")
		contactID := r.URL.Query().Get("contact_id")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "WebSocket Upgrade Failed")
			return
		}

		client := &ws.Client{
			Conn:       conn,
			Send:       make(chan []byte),
			UserID:     userID,
			ContactID:  contactID,
			LocationID: locationID,
			Hub:        hub,
		}

		hub.Register <- client

		// 📨 Flush offline messages on connect + mark delivered
		targetType := "user"
		targetID := userID
		if contactID != "" {
			targetType = "contact"
			targetID = contactID
		}
		// Track presence
		go presence.MarkUserOnline(userID, contactID, locationID)

		go func() {
			<-r.Context().Done()
			presence.MarkUserOffline(userID, contactID, locationID)
		}()

		if offlineMsgs, err := redis.FlushQueuedMessages(targetType, locationID, targetID, func(ids []string) {
			var uuids []uuid.UUID
			for _, id := range ids {
				if uid, err := uuid.Parse(id); err == nil {
					uuids = append(uuids, uid)
				}
			}
			if len(uuids) > 0 {
				if err := messageRepo.MarkMessagesDelivered(uuids); err != nil {
					log.Printf("⚠️ Failed to mark messages as delivered: %v", err)
				} else {
					log.Printf("✅ Marked %d messages as delivered", len(uuids))
				}
			}
		}); err == nil {
			for _, msg := range offlineMsgs {
				client.Send <- msg
			}
		}

		go client.ReadPump()
		go client.WritePump()
		go presence.MarkUserOnline(client.UserID, client.ContactID, client.LocationID)

	}
}

func GetMessageHistory(w http.ResponseWriter, r *http.Request) {
	locationID := r.URL.Query().Get("location_id")
	contactID := r.URL.Query().Get("contact_id")
	userID := r.URL.Query().Get("user_id")

	auth := auth.GetAuthContext(r)
	log.Printf("🔎 GetMessageHistory auth context: userID=%s type=%s", auth.UserID, auth.UserType)

	if locationID == "" || contactID == "" || userID == "" {
		writeError(w, http.StatusBadRequest, "Missing query params")
		return
	}

	if auth.UserType == "DOCTOR" && auth.UserID != userID {
		writeError(w, http.StatusForbidden, "Unauthorized doctor")
		return
	}
	if auth.UserType == "PATIENT" && auth.UserID != contactID {
		writeError(w, http.StatusForbidden, "Unauthorized patient")
		return
	}

	messages, err := messageRepo.GetConversation(locationID, contactID, userID)
	if err != nil {
		log.Printf("❌ Error fetching conversation: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

func MarkMessageAsRead(w http.ResponseWriter, r *http.Request) {
	auth := auth.GetAuthContext(r)
	log.Printf("📝 MarkMessageAsRead called by userID=%s", auth.UserID)

	var payload struct {
		MessageIDs []string `json:"message_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := messageRepo.MarkMessagesRead(payload.MessageIDs)
	if err != nil {
		log.Printf("❌ Failed to mark messages read: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to mark messages as read")
		return
	}

	writeSuccess(w, http.StatusOK, "Messages marked as read")
}

func GetPresenceStatus(w http.ResponseWriter, r *http.Request) {
	locationID := r.URL.Query().Get("location_id")
	userID := r.URL.Query().Get("user_id")       // For checking doctors/staff
	contactID := r.URL.Query().Get("contact_id") // For checking patients

	if locationID == "" || (userID == "" && contactID == "") {
		writeError(w, http.StatusBadRequest, "Missing location_id and user/contact ID")
		return
	}

	status, err := redis.GetPresenceStatus(userID, contactID, locationID)
	if err != nil {
		log.Printf("❌ Presence lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Presence check failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": status, // "online" or "last seen at ..."
	})
}

func UploadChatFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid file upload")
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing file")
		return
	}
	allowedTypes := map[string]bool{
		"image/jpeg":         true,
		"image/png":          true,
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}

	fileType := fileHeader.Header.Get("Content-Type")
	if !allowedTypes[fileType] {
		writeError(w, http.StatusBadRequest, "Unsupported file type")
		return
	}

	url, err := s3.UploadFile(file, fileHeader, "chat-attachments")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"file_url":  url,
		"file_name": fileHeader.Filename,
		"file_type": fileHeader.Header.Get("Content-Type"),
	})
}

func ListChatSessions(repo *repository.MessageRepo) http.HandlerFunc {
	// Supports optional ?location_id, ?limit, and ?offset params
	return func(w http.ResponseWriter, r *http.Request) {
		authCtx := auth.GetAuthContext(r)
		userID := authCtx.UserID
		userType := authCtx.UserType

		if userID == "" || (userType != "DOCTOR" && userType != "PATIENT") {
			writeError(w, http.StatusUnauthorized, "Unauthorized access")
			return
		}

		var contactID string
		if userType == "PATIENT" {
			contactID = userID
		}

		locationID := r.URL.Query().Get("location_id")
		limitParam := r.URL.Query().Get("limit")
		offsetParam := r.URL.Query().Get("offset")

		limit := 20
		offset := 0

		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}

		sessions, err := repo.ListEnrichedChatSessionsWithFilter(userID, contactID, locationID, limit, offset)
		if err != nil {
			log.Printf("❌ Failed to fetch chat sessions: %v", err)
			writeError(w, http.StatusInternalServerError, "Could not fetch sessions")
			return
		}

		writeJSON(w, http.StatusOK, sessions)
	}
}

func SearchMessages(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authCtx := auth.GetAuthContext(r)
		query := r.URL.Query().Get("q")
		locationID := r.URL.Query().Get("location_id")
		limitParam := r.URL.Query().Get("limit")

		if query == "" || locationID == "" {
			writeError(w, http.StatusBadRequest, "Missing required parameters: q, location_id")
			return
		}

		limit := 20
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}

		var contactID string
		if authCtx.UserType == "PATIENT" {
			contactID = authCtx.UserID
		}

		results, err := repo.SearchMessages(authCtx.UserID, contactID, locationID, query, limit)
		if err != nil {
			log.Printf("❌ Failed to search messages: %v", err)
			writeError(w, http.StatusInternalServerError, "Search failed")
			return
		}

		writeJSON(w, http.StatusOK, results)
	}
}

func AdminListSessions(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authCtx := auth.GetAuthContext(r)

		// Only allow users with type ADMIN or SUPERADMIN
		if authCtx.UserType != "ADMIN" && authCtx.UserType != "SUPERADMIN" {
			writeError(w, http.StatusForbidden, "Admin access required")
			return
		}

		locationID := r.URL.Query().Get("location_id")
		limit := 50
		offset := 0

		if val := r.URL.Query().Get("limit"); val != "" {
			if l, err := strconv.Atoi(val); err == nil && l > 0 {
				limit = l
			}
		}
		if val := r.URL.Query().Get("offset"); val != "" {
			if o, err := strconv.Atoi(val); err == nil && o >= 0 {
				offset = o
			}
		}

		sessions, err := repo.AdminListAllSessions(locationID, limit, offset)
		if err != nil {
			log.Printf("❌ AdminListSessions query failed: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to fetch admin sessions")
			return
		}

		writeJSON(w, http.StatusOK, sessions)
	}
}

func AdminDeleteMessages(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authCtx := auth.GetAuthContext(r)
		if authCtx.UserType != "ADMIN" && authCtx.UserType != "SUPERADMIN" {
			writeError(w, http.StatusForbidden, "Unauthorized")
			return
		}

		var payload struct {
			MessageIDs []string `json:"message_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		var uuids []uuid.UUID
		for _, id := range payload.MessageIDs {
			if u, err := uuid.Parse(id); err == nil {
				uuids = append(uuids, u)
			}
		}

		if err := repo.AdminDeleteMessages(uuids); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to delete messages")
			return
		}

		writeSuccess(w, http.StatusOK, "Messages soft-deleted")
	}
}

func DeleteChatMessage(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := auth.GetAuthContext(r)
		if auth.UserType != "ADMIN" && auth.UserType != "DOCTOR" && auth.UserType != "SUPERADMIN" {
			writeError(w, http.StatusForbidden, "Unauthorized")
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid message ID")
			return
		}

		if err := repo.DeleteMessage(id); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to delete message")
			return
		}

		writeSuccess(w, http.StatusOK, "Message deleted")
	}
}

func AddReaction(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := auth.GetAuthContext(r)
		var payload struct {
			MessageID string `json:"message_id"`
			Emoji     string `json:"emoji"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}
		err := repo.AddReaction(payload.MessageID, auth.UserID, payload.Emoji)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to add reaction")
			return
		}
		writeSuccess(w, http.StatusOK, "Reaction added")
	}
}

func RemoveReaction(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := auth.GetAuthContext(r)
		var payload struct {
			MessageID string `json:"message_id"`
			Emoji     string `json:"emoji"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}
		err := repo.RemoveReaction(payload.MessageID, auth.UserID, payload.Emoji)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to remove reaction")
			return
		}
		writeSuccess(w, http.StatusOK, "Reaction removed")
	}
}

func PinMessage(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msgID := chi.URLParam(r, "id")
		err := repo.TogglePinMessage(msgID, true)
		if err != nil {
			log.Printf("❌ Failed to pin message %s: %v", msgID, err)
			writeError(w, http.StatusInternalServerError, "Failed to pin message")
			return
		}
		writeSuccess(w, http.StatusOK, "Message pinned")
	}
}

func UnpinMessage(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msgID := chi.URLParam(r, "id")
		err := repo.TogglePinMessage(msgID, false)
		if err != nil {
			log.Printf("❌ Failed to unpin message %s: %v", msgID, err)
			writeError(w, http.StatusInternalServerError, "Failed to unpin message")
			return
		}
		writeSuccess(w, http.StatusOK, "Message unpinned")
	}
}

func GetPinnedMessages(repo *repository.MessageRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "session_id")
		messages, err := repo.GetPinnedMessages(sessionID)
		if err != nil {
			log.Printf("❌ Failed to fetch pinned messages: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to fetch pinned messages")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}

// POST /auth/device-token
func SaveDeviceToken(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"user_id"`
		Token  string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := messageRepo.UpsertDeviceToken(payload.UserID, payload.Token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save token")
		return
	}

	writeSuccess(w, http.StatusOK, "Device token saved")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("❌ Failed to encode JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, errMsg string) {
	log.Printf("❌ %s", errMsg)
	writeJSON(w, status, map[string]string{"error": errMsg})
}

func writeSuccess(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}
