package redis

import (
	"context"
	"fmt"
	"myGreenMarket/pkg/config"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(config *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", config.Redis.RedisHost, config.Redis.RedisPort),
		Password:     config.Redis.RedisPassword,
		Username:     "default",
		DB:           config.Redis.RedisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// CloseRedisClient closes the Redis connection
func CloseRedisClient(client *redis.Client) error {
	if client != nil {
		return client.Close()
	}

	return nil
}
