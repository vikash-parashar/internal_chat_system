// handlers/chat.go
package handlers

import (
	"encoding/json"
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
	// inside SendMessage handler, after saving the message
	// redis.Publish(msg.LocationID, msg)

	return func(w http.ResponseWriter, r *http.Request) {
		var msg models.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		msg.ID = uuid.New().String()

		if err := messageRepo.SaveMessage(&msg); err != nil {
			http.Error(w, "Could not save message", http.StatusInternalServerError)
			return
		}

		hub.Broadcast <- ws.BroadcastMessage{
			LocationID: msg.LocationID,
			Message:    msg,
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func HandleWebSocket(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		locationID := r.URL.Query().Get("location_id")
		userID := r.URL.Query().Get("user_id")
		contactID := r.URL.Query().Get("contact_id")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "WebSocket Upgrade Failed", http.StatusInternalServerError)
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
		http.Error(w, "Missing query params", http.StatusBadRequest)
		return
	}

	messages, err := messageRepo.GetConversation(locationID, contactID, userID)
	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}

func MarkMessageAsRead(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		MessageIDs []string `json:"message_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := messageRepo.MarkMessagesRead(payload.MessageIDs)
	if err != nil {
		http.Error(w, "Failed to mark messages as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
