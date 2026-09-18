package db

import (
	"context"
	"log"
	"time"

	"github.com/Ashutosh-negi07/live-poll/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient is the shared thread-safe connection pool for all Redis operations.
var RedisClient *redis.Client

// ConnectRedis initialises the Redis client and verifies connectivity with a PING.
func ConnectRedis(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("Connecting to Redis at %s (db: %d) ...", cfg.RedisAddr, cfg.RedisDB)

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// PING → PONG: confirms the Redis server is reachable and authenticated.
	if _, err := client.Ping(ctx).Result(); err != nil {
		return err
	}

	RedisClient = client
	log.Println("Redis connected.")
	return nil
}

// CloseRedis gracefully drains and closes the Redis connection pool.
func CloseRedis() {
	if RedisClient == nil {
		return
	}
	if err := RedisClient.Close(); err != nil {
		log.Printf("Error closing Redis: %v\n", err)
	} else {
		log.Println("Redis disconnected cleanly.")
	}
}
