// redis/pubsub.go
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"internal_chat_system/models"
	"internal_chat_system/ws"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client

func Init(addr, password string, db int) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func Publish(locationID string, msg models.Message) {
	channel := "chat:" + locationID
	data, _ := json.Marshal(msg)
	if err := rdb.Publish(ctx, channel, data).Err(); err != nil {
		log.Println("Redis publish error:", err)
	}
}

func Subscribe(locationID string, hub *ws.Hub) {
	channel := "chat:" + locationID
	pubsub := rdb.Subscribe(ctx, channel)

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var message models.Message
			if err := json.Unmarshal([]byte(msg.Payload), &message); err == nil {
				hub.Broadcast <- ws.BroadcastMessage{
					LocationID: locationID,
					Message:    message,
				}
			}
		}
	}()
}

func QueueOfflineMessage(recipientType, locationID, recipientID string, msg []byte) error {
	key := fmt.Sprintf("offline_queue:%s:%s:%s", recipientType, locationID, recipientID)
	if err := rdb.RPush(ctx, key, msg).Err(); err != nil {
		log.Printf("❌ Failed to queue message in Redis: %v", err)
		return err
	}
	log.Printf("📩 Queued offline message for key %s", key)
	return nil
}

func FlushQueuedMessages(recipientType, locationID, recipientID string, onDeliver func([]string)) ([][]byte, error) {
	log.Printf("📦 Checking offline messages for %s:%s in location %s", recipientType, recipientID, locationID)
	key := fmt.Sprintf("offline_queue:%s:%s:%s", recipientType, locationID, recipientID)
	msgs, err := rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		log.Printf("❌ Failed to fetch offline messages: %v", err)
		return nil, err
	}

	if err := rdb.Del(ctx, key).Err(); err != nil {
		log.Printf("⚠️ Failed to delete offline queue after flush: %v", err)
	}

	var result [][]byte
	var deliveredIDs []string
	for _, msg := range msgs {
		result = append(result, []byte(msg))
		var parsed struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(msg), &parsed); err == nil && parsed.ID != "" {
			deliveredIDs = append(deliveredIDs, parsed.ID)
		}
	}

	log.Printf("📤 Flushed %d offline messages for key %s", len(result), key)
	if onDeliver != nil && len(deliveredIDs) > 0 {
		onDeliver(deliveredIDs)
	}
	log.Printf("✅ Delivered %d offline message(s) to %s:%s", len(result), recipientType, recipientID)
	return result, nil
}

func IsClientConnected(locationID, recipientID string, clients map[string]map[*ws.Client]bool) bool {
	if conns, ok := clients[locationID]; ok {
		for client := range conns {
			if client.UserID == recipientID || client.ContactID == recipientID {
				return true
			}
		}
	}
	return false
}

type PushEvent struct {
	MessageID    string `json:"message_id"`
	LocationID   string `json:"location_id"`
	ReceiverID   string `json:"receiver_id"`   // can be user or contact
	ReceiverType string `json:"receiver_type"` // user or contact
	Content      string `json:"content"`
}

func PublishPushEvent(event PushEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := rdb.Publish(ctx, "push:events", data).Err(); err != nil {
		log.Printf("❌ Failed to publish push event: %v", err)
		return err
	}
	log.Printf("📣 Push event published for receiver %s (%s)", event.ReceiverID, event.ReceiverType)
	return nil
}
