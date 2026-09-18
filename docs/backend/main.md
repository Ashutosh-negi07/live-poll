# Server Entry Point (`backend/main.go`)

## What it does
`main.go` is the bootstrap file for the entire backend. It wires together all the packages (config, databases, router) in the correct order, starts the HTTP server in a background goroutine, and handles graceful shutdown on OS signals.

## Go Concepts Used

### `package main` and `func main()`
Every executable Go program must have exactly one `package main` with a `func main()`. This is the entry point — where execution begins when you run `go run main.go` or `./server`.

### Goroutines (`go func()`)
A goroutine is a lightweight concurrent function managed by the Go runtime. Unlike OS threads, goroutines start with ~2KB of stack and can scale to millions. We start the HTTP server in a goroutine:

```go
go func() {
    server.ListenAndServe()  // blocks forever inside the goroutine
}()
// execution continues here immediately — main goroutine is not blocked
```

Without the `go` keyword, `ListenAndServe()` would block `main()` forever and the shutdown logic below would never run.

### Channels and OS Signals
A channel (`chan`) is a typed pipe for communicating between goroutines.

```go
quit := make(chan os.Signal, 1)       // create a buffered channel that holds 1 signal
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)  // register it for Ctrl+C / docker stop
<-quit  // block here (receive from channel) until a signal arrives
```

When you press `Ctrl+C`, the OS sends `SIGINT` to the process. `signal.Notify` puts that signal into the `quit` channel. `<-quit` unblocks and shutdown begins.

### `defer`
`defer` runs a function call just before the surrounding function returns, regardless of how it exits:

```go
defer db.DisconnectMongo()  // will run when main() exits
defer db.CloseRedis()       // will run when main() exits
```

This guarantees database connections are always cleaned up, even if the server panics.

### `log.Fatalf`
Logs a formatted message and then calls `os.Exit(1)` — exits the process with an error code. Used when a critical dependency (MongoDB, Redis) is unavailable at startup.

---

## The Code

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/Ashutosh-negi07/live-poll/config"
    "github.com/Ashutosh-negi07/live-poll/db"
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
)

func main() {
    cfg := config.LoadConfig()
    gin.SetMode(cfg.GinMode)

    if err := db.ConnectMongo(cfg); err != nil {
        log.Fatalf("Could not connect to MongoDB: %v", err)
    }
    defer db.DisconnectMongo()

    if err := db.ConnectRedis(cfg); err != nil {
        log.Fatalf("Could not connect to Redis: %v", err)
    }
    defer db.CloseRedis()

    router := gin.Default()

    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{cfg.ClientOrigin},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    router.GET("/api/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status":  "ok",
            "message": "live-poll backend is running",
        })
    })

    server := &http.Server{
        Addr:         ":" + cfg.Port,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        log.Printf("Server listening on http://localhost:%s\n", cfg.Port)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v\n", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Forced shutdown: %v\n", err)
    }
    log.Println("Server exited cleanly.")
}
```

## Startup Sequence
```
main() starts
  ↓ config.LoadConfig()   — reads .env into *Config
  ↓ db.ConnectMongo(cfg)  — opens pool, pings MongoDB, fatal if unreachable
  ↓ db.ConnectRedis(cfg)  — opens pool, pings Redis, fatal if unreachable
  ↓ gin.Default()         — creates router with Logger + Recovery middleware built in
  ↓ router.Use(cors)      — attaches CORS middleware
  ↓ router.GET("/api/health", ...) — registers health check route
  ↓ go server.ListenAndServe()     — starts HTTP server in background goroutine
  ↓ <-quit                — main goroutine blocks here waiting for Ctrl+C / SIGTERM
  ↓ server.Shutdown(ctx)  — stops accepting new requests, waits 5s for active ones
  ↓ deferred CloseRedis() — closes Redis pool
  ↓ deferred DisconnectMongo() — closes MongoDB pool
  ↓ process exits cleanly
```

## CORS Configuration
CORS (Cross-Origin Resource Sharing) tells the browser which frontend origins are allowed to call our API. Without it, the browser blocks every request from `localhost:5173` to `localhost:8080` because they are on different ports (different "origins").

- `AllowOrigins`: Only `http://localhost:5173` (our React dev server) is permitted.
- `AllowHeaders`: `Authorization` must be listed so JWT bearer tokens are allowed through.
- `MaxAge: 12h`: Browser caches the preflight approval for 12 hours — avoids a redundant `OPTIONS` request before every API call.

## How to Run
```bash
cd backend
go run main.go
```
Expected output:
```
2026/09/18 18:00:00 Connecting to MongoDB at mongodb://localhost:27017 ...
2026/09/18 18:00:00 MongoDB connected  (database: livepoll)
2026/09/18 18:00:00 Connecting to Redis at localhost:6379 (db: 0) ...
2026/09/18 18:00:00 Redis connected.
[GIN-debug] GET    /api/health
2026/09/18 18:00:00 Server listening on http://localhost:8080
```
