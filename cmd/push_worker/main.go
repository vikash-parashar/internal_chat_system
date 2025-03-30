package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

type PushEvent struct {
	MessageID    string `json:"message_id"`
	LocationID   string `json:"location_id"`
	ReceiverID   string `json:"receiver_id"`
	ReceiverType string `json:"receiver_type"`
	Content      string `json:"content"`
}

var ctx = context.Background()

func main() {

	// err := fcm.Init("path/to/your/firebase-adminsdk.json")
	// if err != nil {
	// 	log.Fatalf("❌ FCM Init failed: %v", err)
	// }

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	pubsub := rdb.Subscribe(ctx, "push:events")
	log.Println("📡 Listening for push events on Redis channel: push:events")

	ch := pubsub.Channel()
	for msg := range ch {
		var event PushEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("❌ Failed to parse push event: %v", err)
			continue
		}

		// 🔔 Simulated push action (replace this with Firebase/Twilio/Mailgun/etc.)
		log.Printf("🔔 New push: [%s] -> %s:%s — \"%s\"",
			event.MessageID, event.ReceiverType, event.ReceiverID, event.Content,
		)

		// TODO: SendPushNotification(event)
	}
}
