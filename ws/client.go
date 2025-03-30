package ws

import (
	"encoding/json"
	"log"
	"time"

	"internal_chat_system/models"
	"internal_chat_system/presence"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn       *websocket.Conn
	Send       chan []byte
	UserID     string
	ContactID  string
	LocationID string
	Hub        *Hub
}

func (c *Client) ReadPump() {
	defer func() {
		// ❌ Mark user offline in Redis
		presence.MarkUserOffline(c.UserID, c.ContactID, c.LocationID)

		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	// ✅ Mark user online on connect
	presence.MarkUserOnline(c.UserID, c.ContactID, c.LocationID)

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Detect base type
		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg, &base); err != nil {
			continue
		}

		switch base.Type {
		case "typing":
			// Broadcast raw typing event to everyone in the same location
			c.Hub.Broadcast <- BroadcastMessage{
				LocationID: c.LocationID,
				RawData:    msg,
			}
		case "ping":
			// Refresh online status heartbeat
			presence.MarkUserOnline(c.UserID, c.ContactID, c.LocationID)
		default:
			// Unknown type, ignore
		}
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := c.Conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Println("Write error:", err)
				return
			}
		}
	}
}

func EncodeMessage(msg models.Message) ([]byte, error) {
	return json.Marshal(msg)
}
