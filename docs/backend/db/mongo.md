# MongoDB Client (`backend/db/mongo.go`)

## What it does
Opens and manages a shared connection pool to MongoDB. Called once at server startup from `main.go`. Provides `GetCollection("name")` for handlers to read and write documents, and `DisconnectMongo()` for clean shutdown.

## Key Concepts

### Connection Pool
A connection pool is a set of pre-opened, reusable TCP connections to a database. Instead of opening a new connection for every HTTP request (slow, expensive), all goroutines share a pool of persistent connections. `*mongo.Client` manages this pool automatically.

### `context.Context` and Timeouts
A `context.Context` carries a deadline, cancellation signal, or request-scoped values across API boundaries.

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```
- `context.WithTimeout(...)` creates a context that auto-cancels after 10 seconds.
- `defer cancel()` ensures the internal timer is cleaned up when the function returns — prevents a goroutine leak.
- This context is passed to `client.Ping(ctx, ...)` — if MongoDB doesn't respond within 10 seconds, the ping fails immediately with an error instead of hanging forever.

### Why Ping?
`mongo.Connect()` does not actually open a TCP connection — it just configures the client. The real connection happens lazily on the first operation. Calling `client.Ping()` forces an immediate connection attempt, so the server **fails fast** at startup rather than discovering the database is down on the first user request.

### Package-level variables
```go
var MongoClient *mongo.Client
var MongoDB *mongo.Database
```
Declared at the package level so they are accessible from any file inside `package db`. `main.go` calls `ConnectMongo()` which sets them; `handlers` package calls `db.GetCollection()` which uses `MongoDB`.

---

## The Code

```go
package db

import (
    "context"
    "log"
    "time"

    "github.com/Ashutosh-negi07/live-poll/config"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
    "go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var MongoClient *mongo.Client
var MongoDB *mongo.Database

func ConnectMongo(cfg *config.Config) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    log.Printf("Connecting to MongoDB at %s ...", cfg.MongoURI)

    client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
    if err != nil {
        return err
    }

    if err := client.Ping(ctx, readpref.Primary()); err != nil {
        return err
    }

    MongoClient = client
    MongoDB = client.Database(cfg.MongoDBName)

    log.Printf("MongoDB connected  (database: %s)\n", cfg.MongoDBName)
    return nil
}

func GetCollection(name string) *mongo.Collection {
    return MongoDB.Collection(name)
}

func DisconnectMongo() {
    if MongoClient == nil {
        return
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := MongoClient.Disconnect(ctx); err != nil {
        log.Printf("Error disconnecting MongoDB: %v\n", err)
    } else {
        log.Println("MongoDB disconnected cleanly.")
    }
}
```

## How handlers use it
```go
// Inside any handler file:
import "github.com/Ashutosh-negi07/live-poll/db"

collection := db.GetCollection("polls")
result, err := collection.InsertOne(ctx, pollDocument)
```

## Error Handling Pattern
All functions return `error`. `main.go` checks the returned error and calls `log.Fatalf(...)` if either database connection fails — this immediately exits the process with a non-zero status code, which Docker/cloud platforms detect and alert on.
