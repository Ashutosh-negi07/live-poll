package db

import (
	"context"
	"fmt"
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

	var opts *redis.Options
	if cfg.RedisURL != "" {
		var err error
		opts, err = redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return fmt.Errorf("invalid REDIS_URL: %w", err)
		}
		log.Printf("Connecting to Redis via REDIS_URL at %s ...", opts.Addr)
	} else {
		log.Printf("Connecting to Redis at %s (db: %d) ...", cfg.RedisAddr, cfg.RedisDB)
		opts = &redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}
	}

	client := redis.NewClient(opts)

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
