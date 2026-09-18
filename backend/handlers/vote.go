package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Ashutosh-negi07/live-poll/db"
	"github.com/Ashutosh-negi07/live-poll/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Vote handles POST /api/polls/:id/vote  (public — no JWT needed)
func Vote(c *gin.Context) {
	pollID := c.Param("id")

	// 1. Bind and validate request body
	var input models.VoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// 4. Generate a voter fingerprint: SHA256(IP + UserAgent)
	// This is hashed so we never store raw IPs — privacy-safe deduplication.
	raw := c.ClientIP() + "|" + c.GetHeader("User-Agent")
	hash := sha256.Sum256([]byte(raw))
	voterKey := fmt.Sprintf("%x", hash)

	// 5. Deduplicate: SADD returns 1 if added (new voter), 0 if already exists
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
		// Rollback the deduplication entry so the user can retry
		db.RedisClient.SRem(ctx, votersKey, voterKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record vote"})
		return
	}

	// 7. Fetch updated counts and publish to all SSE subscribers
	rawVotes, _ := db.RedisClient.HGetAll(ctx, votesKey).Result()
	if payload, err := json.Marshal(rawVotes); err == nil {
		db.RedisClient.Publish(ctx, "poll:"+pollID+":live", string(payload))
	}

	// 8. Persist vote log to MongoDB asynchronously (non-blocking)
	// The HTTP response is sent immediately; the DB write happens in the background.
	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer bgCancel()

		log := models.VoteLog{
			ID:        bson.NewObjectID(),
			PollID:    objID,
			OptionID:  input.OptionID,
			VoterHash: voterKey,
			CreatedAt: time.Now().UTC(),
		}
		if _, err := db.GetCollection("vote_logs").InsertOne(bgCtx, log); err != nil {
			fmt.Printf("Warning: failed to write vote_log for poll %s: %v\n", pollID, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "vote recorded"})
}
