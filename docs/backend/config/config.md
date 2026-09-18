# Config Loader (`backend/config/config.go`)

## What it does
Reads all environment variables once at server startup and returns a single strongly-typed `*Config` struct. Every other package in the backend receives this struct and reads values like `cfg.Port` or `cfg.JWTSecret` — no package ever calls `os.Getenv()` directly.

## Go Concepts Used

### Structs
A struct is a named collection of typed fields — Go's equivalent of an object shape. All field names start with a capital letter so they are exported (accessible from other packages like `db` and `handlers`).

```go
type Config struct {
    Port    string  // exported — other packages can read cfg.Port
    RedisDB int     // exported — used in redis.go
}
```

### Pointers (`*Config`)
`LoadConfig()` returns `*Config` — a pointer to the struct (a memory address), not a copy. This is efficient: passing a pointer around is always 8 bytes regardless of how many fields `Config` has.

```go
cfg := config.LoadConfig()  // cfg is *Config
fmt.Println(cfg.Port)       // dereference happens automatically in Go
```

### Unexported helper functions
`getEnv` and `getEnvAsInt` start with lowercase — they are private to the `config` package. External packages cannot call them directly, only `LoadConfig()` is public.

### `strconv.Atoi`
Environment variables are always strings. `REDIS_DB=0` arrives as `"0"`. `strconv.Atoi` converts it to the integer `0`. If conversion fails (e.g. someone wrote `REDIS_DB=abc`), the function logs a warning and falls back to the default instead of crashing.

---

## The Code

```go
package config

import (
    "log"
    "os"
    "strconv"

    "github.com/joho/godotenv"
)

type Config struct {
    Port           string
    GinMode        string
    ClientOrigin   string
    MongoURI       string
    MongoDBName    string
    RedisAddr      string
    RedisPassword  string
    RedisDB        int
    JWTSecret      string
    JWTExpiryHours int
}

func LoadConfig() *Config {
    if err := godotenv.Load(); err != nil {
        log.Println("Note: .env file not found, reading from system environment")
    }

    return &Config{
        Port:           getEnv("PORT", "8080"),
        GinMode:        getEnv("GIN_MODE", "debug"),
        ClientOrigin:   getEnv("CLIENT_ORIGIN", "http://localhost:5173"),
        MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
        MongoDBName:    getEnv("MONGO_DB_NAME", "livepoll"),
        RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
        RedisPassword:  getEnv("REDIS_PASSWORD", ""),
        RedisDB:        getEnvAsInt("REDIS_DB", 0),
        JWTSecret:      getEnv("JWT_SECRET", "change_this_in_production"),
        JWTExpiryHours: getEnvAsInt("JWT_EXPIRY_HOURS", 24),
    }
}

func getEnv(key, fallback string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return fallback
}

func getEnvAsInt(key string, fallback int) int {
    valStr := os.Getenv(key)
    if valStr == "" {
        return fallback
    }
    val, err := strconv.Atoi(valStr)
    if err != nil {
        log.Printf("Warning: %s='%s' is not a valid integer. Using default: %d\n", key, valStr, fallback)
        return fallback
    }
    return val
}
```

## How It Flows
```
main.go calls config.LoadConfig()
    → godotenv.Load() reads backend/.env, calls os.Setenv for each line
    → getEnv("PORT", "8080") calls os.Getenv("PORT") → returns "8080"
    → getEnvAsInt("REDIS_DB", 0) calls os.Getenv("REDIS_DB") → "0" → Atoi → 0
    → &Config{...} builds the struct and returns its memory address
main.go stores it as cfg := config.LoadConfig()
main.go passes cfg to db.ConnectMongo(cfg), db.ConnectRedis(cfg), etc.
```
