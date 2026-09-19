package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Ashutosh-negi07/live-poll/config"
	"github.com/Ashutosh-negi07/live-poll/db"
	"github.com/Ashutosh-negi07/live-poll/middleware"
	"github.com/Ashutosh-negi07/live-poll/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Vote handles POST /api/polls/:id/vote  (public — no JWT required, but uses it if present)
func Vote(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollID := c.Param("id")

		// 1. Bind and validate request body
		var input models.VoteInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": friendlyError(err)})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 2. Verify poll exists and is still active
		objID, err := bson.ObjectIDFromHex(pollID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
			return
		}

		var poll models.Poll
		if err := db.GetCollection("polls").FindOne(ctx, bson.M{"_id": objID}).Decode(&poll); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
			return
		}

		if !poll.IsActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "this poll is closed"})
			return
		}

		// 3. Verify the submitted option actually exists in this poll
		validOption := false
		for _, opt := range poll.Options {
			if opt.ID == input.OptionID {
				validOption = true
				break
			}
		}
		if !validOption {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid option id"})
			return
		}

		// 4. Determine voter identity for deduplication.
		//    If the request carries a valid JWT (logged-in user), use their user ID —
		//    this means different accounts on the same device can vote independently.
		//    If no JWT (anonymous visitor), fall back to SHA256(IP + UserAgent).
		voterKey := voterFingerprint(c, cfg)

		// 5. Deduplicate: SADD returns 1 if new voter, 0 if already voted
		votersKey := "poll:" + pollID + ":voters"
		added, err := db.RedisClient.SAdd(ctx, votersKey, voterKey).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process vote"})
			return
		}
		if added == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "you have already voted on this poll"})
			return
		}

		// 6. Atomically increment the vote counter in Redis
		votesKey := "poll:" + pollID + ":votes"
		if err := db.RedisClient.HIncrBy(ctx, votesKey, input.OptionID, 1).Err(); err != nil {
			db.RedisClient.SRem(ctx, votersKey, voterKey) // rollback
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record vote"})
			return
		}

		// 7. Fetch updated counts and publish to all SSE subscribers
		rawVotes, _ := db.RedisClient.HGetAll(ctx, votesKey).Result()
		if payload, err := json.Marshal(rawVotes); err == nil {
			db.RedisClient.Publish(ctx, "poll:"+pollID+":live", string(payload))
		}

		c.JSON(http.StatusOK, gin.H{"message": "vote recorded"})
	}
}

// voterFingerprint returns a unique key for deduplication.
// If the request includes a valid JWT, returns "user:<userID>" so that
// different accounts on the same device are treated as different voters.
// Otherwise falls back to SHA256(IP + UserAgent).
func voterFingerprint(c *gin.Context, cfg *config.Config) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			claims := &middleware.Claims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err == nil && token.Valid && claims.UserID != "" {
				return "user:" + claims.UserID
			}
		}
	}
	// Anonymous fallback
	raw := c.ClientIP() + "|" + c.GetHeader("User-Agent")
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("anon:%x", hash)
}
