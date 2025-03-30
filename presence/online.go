package presence

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client // 👈 define redis client here

// Init initializes the Redis client used for presence tracking
func Init(redisClient *redis.Client) {
	rdb = redisClient
}

func MarkUserOnline(userID, contactID, locationID string) {
	ctx := context.Background()

	if userID != "" {
		key := fmt.Sprintf("online:user:%s", userID)
		rdb.Set(ctx, key, "1", time.Minute)
		rdb.Set(ctx, fmt.Sprintf("last_seen:user:%s", userID), time.Now().Unix(), 0)
	}

	if contactID != "" {
		key := fmt.Sprintf("online:contact:%s", contactID)
		rdb.Set(ctx, key, "1", time.Minute)
		rdb.Set(ctx, fmt.Sprintf("last_seen:contact:%s", contactID), time.Now().Unix(), 0)
	}
}

func MarkUserOffline(userID, contactID, locationID string) {
	ctx := context.Background()

	if userID != "" {
		rdb.Del(ctx, fmt.Sprintf("online:user:%s", userID))
	}
	if contactID != "" {
		rdb.Del(ctx, fmt.Sprintf("online:contact:%s", contactID))
	}
}

func IsUserOnline(userID string) bool {
	ctx := context.Background()
	val, err := rdb.Get(ctx, fmt.Sprintf("online:user:%s", userID)).Result()
	return err == nil && val == "1"
}

func GetLastSeen(userID string) (int64, error) {
	ctx := context.Background()
	return rdb.Get(ctx, fmt.Sprintf("last_seen:user:%s", userID)).Int64()
}
