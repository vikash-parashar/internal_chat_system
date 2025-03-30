// handlers/chat.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"internal_chat_system/models"
	"internal_chat_system/repository"
	"internal_chat_system/ws"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var messageRepo *repository.MessageRepo

func Init(repo *repository.MessageRepo) {
	messageRepo = repo
}

func SendMessage(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg models.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
		msg.ID = uuid.New().String()

		if msg.LocationID == "" || msg.SenderUserID == "" || msg.ReceiverContactID == "" || msg.Content == "" {
			log.Printf("❌ Validation error: missing fields in %+v", msg)
			writeError(w, http.StatusBadRequest, "Missing required fields")
			return
		}

		if err := messageRepo.SaveMessage(&msg); err != nil {
			log.Printf("❌ DB Error on SaveMessage: %v", err)
			writeError(w, http.StatusInternalServerError, "Could not save message")
			return
		}

		hub.Broadcast <- ws.BroadcastMessage{
			LocationID: msg.LocationID,
			Message:    msg,
		}

		writeJSON(w, http.StatusCreated, msg)
	}
}

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

		go client.ReadPump()
		go client.WritePump()
	}
}

func GetMessageHistory(w http.ResponseWriter, r *http.Request) {
	locationID := r.URL.Query().Get("location_id")
	contactID := r.URL.Query().Get("contact_id")
	userID := r.URL.Query().Get("user_id")

	if locationID == "" || contactID == "" || userID == "" {
		writeError(w, http.StatusBadRequest, "Missing query params")
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
