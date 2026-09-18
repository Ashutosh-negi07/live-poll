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

	// ── 5. Create the Gin router ─────────────────────────────────────────────
	router := gin.Default()

	// ── 6. Apply CORS middleware ─────────────────────────────────────────────
	// This allows our React frontend on port 5173 to call our Go API on 8080.
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.ClientOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── 7. Health check endpoint ─────────────────────────────────────────────
	// A quick way to verify the server + both databases are alive.
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "live-poll backend is running",
		})
	})

	// ── 8. Configure the HTTP server ─────────────────────────────────────────
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── 9. Start server in a background goroutine ────────────────────────────
	// ListenAndServe blocks forever, so we run it in a goroutine so the main
	// goroutine can continue to the shutdown signal listener below.
	go func() {
		log.Printf("Server listening on http://localhost:%s\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	// ── 10. Graceful shutdown ────────────────────────────────────────────────
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
