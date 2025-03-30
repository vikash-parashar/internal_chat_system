// ws/hub.go
package ws

import (
	"internal_chat_system/models"
	"log"
)

type BroadcastMessage struct {
	LocationID string
	Message    models.Message
	RawData    []byte
}

type Hub struct {
	Clients    map[string]map[*Client]bool // locationID -> clients
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan BroadcastMessage
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			if h.Clients[client.LocationID] == nil {
				h.Clients[client.LocationID] = make(map[*Client]bool)
				log.Printf("✅ New location group created: %s", client.LocationID)
			}
			h.Clients[client.LocationID][client] = true
			log.Printf("👤 Client registered: user=%s contact=%s location=%s", client.UserID, client.ContactID, client.LocationID)

		case client := <-h.Unregister:
			if clients, ok := h.Clients[client.LocationID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					log.Printf("👋 Client unregistered: user=%s contact=%s location=%s", client.UserID, client.ContactID, client.LocationID)
				}
			}

		case msg := <-h.Broadcast:
			clients, ok := h.Clients[msg.LocationID]
			if !ok {
				log.Printf("⚠️ No clients to broadcast for location %s", msg.LocationID)
				continue
			}
			data, err := EncodeMessage(msg.Message)
			if err != nil {
				log.Printf("❌ Failed to encode message for broadcast: %v", err)
				continue
			}
			for client := range clients {
				select {
				case client.Send <- data:
					log.Printf("📤 Message sent to client: user=%s contact=%s", client.UserID, client.ContactID)
				default:
					close(client.Send)
					delete(clients, client)
					log.Printf("⚠️ Client channel closed unexpectedly: user=%s contact=%s", client.UserID, client.ContactID)
				}
			}
			if msg.RawData != nil {
				for client := range h.Clients[msg.LocationID] {
					client.Send <- msg.RawData
				}
				continue
			}
		}

	}
}
