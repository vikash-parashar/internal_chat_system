// redis/pubsub.go
package redis

import (
	"context"
	"encoding/json"
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
