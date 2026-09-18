package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Ashutosh-negi07/live-poll/config"
	"github.com/Ashutosh-negi07/live-poll/db"
	"github.com/Ashutosh-negi07/live-poll/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// CreatePoll handles POST /api/polls  (protected)
func CreatePoll(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get creator identity from JWT context
		userID := c.MustGet("userID").(string)
		name := c.MustGet("name").(string)

		// 2. Bind and validate request body
		var input models.CreatePollInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 3. Convert creator string ID to ObjectID
		creatorObjID, err := bson.ObjectIDFromHex(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		// 4. Build options slice with generated IDs (opt_0, opt_1, ...)
		options := make([]models.Option, len(input.Options))
		for i, text := range input.Options {
			options[i] = models.Option{
				ID:   fmt.Sprintf("opt_%d", i),
				Text: text,
			}
		}

		// 5. Build poll document
		poll := models.Poll{
			ID:          bson.NewObjectID(),
			Title:       input.Title,
			Description: input.Description,
			Options:     options,
			CreatorID:   creatorObjID,
			CreatorName: name,
			IsActive:    true,
			CreatedAt:   time.Now().UTC(),
		}

		// 6. Insert into MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := db.GetCollection("polls").InsertOne(ctx, poll); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create poll"})
			return
		}

		// 7. Seed Redis hash with zero counts for every option
		// Key: "poll:<id>:votes"  Value: { "opt_0": 0, "opt_1": 0, ... }
		pollKey := "poll:" + poll.ID.Hex() + ":votes"
		args := make([]interface{}, 0, len(options)*2)
		for _, opt := range options {
			args = append(args, opt.ID, 0)
		}
		if err := db.RedisClient.HSet(ctx, pollKey, args...).Err(); err != nil {
			// Non-fatal: SSE stream can still work; log and continue
			fmt.Printf("Warning: could not seed Redis for poll %s: %v\n", poll.ID.Hex(), err)
		}

		c.JSON(http.StatusCreated, poll)
	}
}

// GetPoll handles GET /api/polls/:id  (public — anyone with the link)
func GetPoll(c *gin.Context) {
	pollID := c.Param("id")

	objID, err := bson.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Fetch poll structure from MongoDB
	var poll models.Poll
	if err := db.GetCollection("polls").FindOne(ctx, bson.M{"_id": objID}).Decode(&poll); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return
	}

	// 2. Fetch live vote counts from Redis
	rawVotes, _ := db.RedisClient.HGetAll(ctx, "poll:"+pollID+":votes").Result()

	votes := make(map[string]int64)
	var total int64
	for optID, countStr := range rawVotes {
		count, _ := strconv.ParseInt(countStr, 10, 64)
		votes[optID] = count
		total += count
	}

	// 3. Return merged MongoDB + Redis result
	c.JSON(http.StatusOK, models.PollResult{
		Poll:       poll,
		Votes:      votes,
		TotalVotes: total,
	})
}

// ListMyPolls handles GET /api/polls/my  (protected)
func ListMyPolls(c *gin.Context) {
	userID := c.MustGet("userID").(string)

	creatorObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all polls created by this user, newest first
	cursor, err := db.GetCollection("polls").Find(ctx, bson.M{"creator_id": creatorObjID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch polls"})
		return
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode polls"})
		return
	}

	// Return empty array instead of null when no polls exist
	if polls == nil {
		polls = []models.Poll{}
	}

	c.JSON(http.StatusOK, polls)
}

// ClosePoll handles PATCH /api/polls/:id/close  (protected — creator only)
func ClosePoll(c *gin.Context) {
	pollID := c.Param("id")
	userID := c.MustGet("userID").(string)

	objID, err := bson.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
		return
	}

	creatorObjID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Only update the poll if BOTH _id AND creator_id match — prevents one user closing another's poll
	filter := bson.M{"_id": objID, "creator_id": creatorObjID}
	update := bson.M{"$set": bson.M{"is_active": false}}

	result, err := db.GetCollection("polls").UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not close poll"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found or you are not the creator"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "poll closed successfully"})
}
