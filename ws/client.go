package ws

import (
	"encoding/json"
	"log"
	"time"

	"internal_chat_system/models"

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
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
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
