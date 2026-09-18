# Redis Client (`backend/db/redis.go`)

## What it does
Opens and manages a shared thread-safe connection pool to Redis. Called once at server startup from `main.go`. Exposes `RedisClient` for all Redis operations throughout the app (vote counting, deduplication, Pub/Sub broadcasting), and `CloseRedis()` for clean shutdown.

## Key Concepts

### Why Redis for this project?
Redis is an in-memory data store — all data lives in RAM. This gives it sub-millisecond response times that no disk-backed database (MongoDB, PostgreSQL) can match. We use it for three specific tasks:

| Command | Purpose |
|:---|:---|
| `HINCRBY poll:<id>:votes <option> 1` | Atomic vote counter — no race conditions |
| `SADD poll:<id>:voters <voter_key>` | Duplicate vote prevention in O(1) |
| `PUBLISH poll:<id>:live <payload>` | Real-time broadcast to all SSE stream handlers |

### Thread Safety
`*redis.Client` from `go-redis/v9` is **safe for concurrent use by multiple goroutines**. Every incoming HTTP request runs in its own goroutine, and all of them safely share the single `RedisClient` variable.

### `PING` / `PONG`
The simplest Redis health check. The client sends `PING`, the server replies `PONG`. If the connection, authentication, or network fails, `.Ping()` returns an error immediately.

---

## The Code

```go
package db

import (
    "context"
    "log"
    "time"

    "github.com/Ashutosh-negi07/live-poll/config"
    "github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis(cfg *config.Config) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    log.Printf("Connecting to Redis at %s (db: %d) ...", cfg.RedisAddr, cfg.RedisDB)

    client := redis.NewClient(&redis.Options{
        Addr:     cfg.RedisAddr,
        Password: cfg.RedisPassword,
        DB:       cfg.RedisDB,
    })

    if _, err := client.Ping(ctx).Result(); err != nil {
        return err
    }

    RedisClient = client
    log.Println("Redis connected.")
    return nil
}

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
```

## How handlers use it
```go
// Inside vote handler — atomic increment:
db.RedisClient.HIncrBy(ctx, "poll:"+pollID+":votes", optionID, 1)

// Inside vote handler — duplicate check:
added, _ := db.RedisClient.SAdd(ctx, "poll:"+pollID+":voters", voterKey).Result()

// Inside stream handler — subscribe to live updates:
pubsub := db.RedisClient.Subscribe(ctx, "poll:"+pollID+":live")
```

## Difference from `mongo.go`
`CloseRedis()` does not need a timeout context because Redis closes its connection pool synchronously and immediately — there are no in-flight disk write operations to wait for, unlike MongoDB.
