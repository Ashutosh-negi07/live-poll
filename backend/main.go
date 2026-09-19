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
	"github.com/Ashutosh-negi07/live-poll/handlers"
	"github.com/Ashutosh-negi07/live-poll/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	// ── 1. Load configuration ────────────────────────────────────────────────
	cfg := config.LoadConfig()

	// ── 2. Set Gin mode (debug prints routes; release is silent & faster) ────
	gin.SetMode(cfg.GinMode)

	// ── 3. Connect to MongoDB (fatal if it fails — nothing works without it) ─
	if err := db.ConnectMongo(cfg); err != nil {
		log.Fatalf("Could not connect to MongoDB: %v", err)
	}
	defer db.DisconnectMongo()

	// ── 4. Connect to Redis (fatal if it fails — real-time engine needs it) ──
	if err := db.ConnectRedis(cfg); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	defer db.CloseRedis()

	// ── 5. Ensure MongoDB indexes ─────────────────────────────────────────────
	// Indexes are created idempotently — safe to run on every startup.
	// The unique index on users.email is the authoritative guard against
	// duplicate accounts (application-level check is a fast-fail, not sufficient).
	ensureIndexes()

	// ── 6. Create the Gin router ──────────────────────────────────────────────
	router := gin.Default()

	// ── 7. Apply CORS middleware ──────────────────────────────────────────────
	// This allows our React frontend on port 5173 to call our Go API on 8080.
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.ClientOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── 8. Health check endpoint ──────────────────────────────────────────────
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "live-poll backend is running",
		})
	})

	// ── 9. Public routes (no JWT required) ───────────────────────────────────
	router.POST("/api/auth/register", handlers.Register(cfg))
	router.POST("/api/auth/login", handlers.Login(cfg))
	router.GET("/api/polls/:id", handlers.GetPoll)
	router.POST("/api/polls/:id/vote", handlers.Vote(cfg))   // uses JWT if present
	router.GET("/api/polls/:id/stream", handlers.Stream)

	// ── 10. Protected routes (JWT required) ──────────────────────────────────
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		protected.GET("/auth/me", handlers.GetMe)
		protected.POST("/polls", handlers.CreatePoll)
		protected.GET("/polls/my", handlers.ListMyPolls)
		protected.PATCH("/polls/:id/close", handlers.ClosePoll)
		protected.DELETE("/polls/:id", handlers.DeletePoll)
	}

	// ── 11. Configure the HTTP server ────────────────────────────────────────
	server := &http.Server{
		Addr:        ":" + cfg.Port,
		Handler:     router,
		ReadTimeout: 15 * time.Second,
		// WriteTimeout is intentionally omitted — SSE connections must stay
		// open indefinitely and a global write deadline would kill them.
		IdleTimeout: 60 * time.Second,
	}

	// ── 12. Start server in a background goroutine ────────────────────────────
	// ListenAndServe blocks forever, so we run it in a goroutine so the main
	// goroutine can continue to the shutdown signal listener below.
	go func() {
		log.Printf("Server listening on http://localhost:%s\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	// ── 13. Graceful shutdown ─────────────────────────────────────────────────
	// Block here until the OS sends SIGINT (Ctrl+C) or SIGTERM (docker stop).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give any in-flight requests up to 5 seconds to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown: %v\n", err)
	}
	log.Println("Server exited cleanly.")
}

// ensureIndexes creates all required MongoDB indexes at startup.
// MongoDB's CreateIndex is idempotent — calling it on an existing index is a no-op.
func ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Unique index on users.email
	// Enforces one account per email at the database level — prevents race conditions
	// where two simultaneous registrations with the same email both pass the app-level check.
	_, err := db.GetCollection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("Warning: could not create users.email index: %v", err)
	} else {
		log.Println("Index ensured: users.email (unique)")
	}

	// 2. Index on polls.creator_id
	// Speeds up GET /api/polls/my which filters all polls by the logged-in user's ID.
	_, err = db.GetCollection("polls").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "creator_id", Value: 1}},
	})
	if err != nil {
		log.Printf("Warning: could not create polls.creator_id index: %v", err)
	} else {
		log.Println("Index ensured: polls.creator_id")
	}

}
