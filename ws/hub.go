package ws

import "internal_chat_system/models"

type BroadcastMessage struct {
	LocationID string
	Message    models.Message
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
			}
			h.Clients[client.LocationID][client] = true

		case client := <-h.Unregister:
			if clients, ok := h.Clients[client.LocationID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
				}
			}

		case msg := <-h.Broadcast:
			if clients, ok := h.Clients[msg.LocationID]; ok {
				data, _ := EncodeMessage(msg.Message)
				for client := range clients {
					select {
					case client.Send <- data:
					default:
						close(client.Send)
						delete(clients, client)
					}
				}
			}
		}
	}
}
