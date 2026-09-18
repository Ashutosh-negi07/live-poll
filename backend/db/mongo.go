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

// MongoClient is the shared connection pool used across all handlers.
var MongoClient *mongo.Client

// MongoDB is the specific database instance ("livepoll") we operate on.
var MongoDB *mongo.Database

// ConnectMongo initialises the MongoDB client and verifies connectivity with a Ping.
func ConnectMongo(cfg *config.Config) error {
	// WithTimeout gives MongoDB exactly 10 seconds to respond before we abort.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("Connecting to MongoDB at %s ...", cfg.MongoURI)

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return err
	}

	// Ping confirms a real TCP connection was made to the primary node.
	// Without this, mongo.Connect succeeds even if the server is offline.
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}

	MongoClient = client
	MongoDB = client.Database(cfg.MongoDBName)

	log.Printf("MongoDB connected  (database: %s)\n", cfg.MongoDBName)
	return nil
}

// GetCollection returns a handle to a named collection inside our database.
// Example: db.GetCollection("users"), db.GetCollection("polls")
func GetCollection(name string) *mongo.Collection {
	return MongoDB.Collection(name)
}

// DisconnectMongo gracefully closes the connection pool during server shutdown.
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
