package redis

import (
	"fmt"

	"github.com/go-redis/redis"
)

const onlineTTL = 60 // seconds

func GetPresenceStatus(userID, contactID, locationID string) (string, error) {
	id := getPresenceKey(userID, contactID, locationID)
	val, err := rdb.Get(ctx, id).Result()
	if err == redis.Nil {
		return "offline", nil
	} else if err != nil {
		return "", err
	}
	if val == "online" {
		return "online", nil
	}
	return fmt.Sprintf("last seen at %s", val), nil
}

func getPresenceKey(userID, contactID, locationID string) string {
	id := userID
	if userID == "" {
		id = contactID
	}
	return fmt.Sprintf("presence:%s:%s", locationID, id)
}
