package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const redisAddr = "localhost:6379"

// OpenRedis opens the applicaiton Redis connection and returns the client instance.
func OpenRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	return client
}

// PingRedis pings the Redis server to check if it's reachable.
func PingRedis(client *redis.Client) error {
	if err := client.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

// InitializeGroup initializes the Redis consumer group for job processing.
func InitializeGroup(client *redis.Client) error {

	// Create a consumer group for job processing if it doesn't exist.
	err := client.XGroupCreateMkStream(context.Background(), "job_queue", "job_workers", "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create Redis consumer group: %w", err)
	}

	return nil
}
