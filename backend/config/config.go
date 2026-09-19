package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration values for the application.
type Config struct {
	Port           string
	GinMode        string
	ClientOrigin   string
	MongoURI       string
	MongoDBName    string
	RedisURL       string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	JWTSecret      string
	JWTExpiryHours int
}

// LoadConfig reads from .env (if it exists) then from system environment variables.
// It returns a *Config with safe fallback defaults for any unset variable.
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
		RedisURL:       getEnv("REDIS_URL", ""),
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
