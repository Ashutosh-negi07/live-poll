package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Ashutosh-negi07/live-poll/db"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Stream handles GET /api/polls/:id/stream  (public — Server-Sent Events)
// Keeps the HTTP connection open and pushes live vote updates to the client.
func Stream(c *gin.Context) {
	pollID := c.Param("id")

	// 1. Verify poll exists before opening a persistent connection
	objID, err := bson.ObjectIDFromHex(pollID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid poll id"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var pollCheck bson.M
	if err := db.GetCollection("polls").FindOne(ctx, bson.M{"_id": objID}).Decode(&pollCheck); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return
	}

	// 2. Set SSE headers — must be set before any body is written
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disables Nginx proxy buffering

	// 3. Get the http.Flusher — required to push data without closing the connection
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// 4. Send current vote snapshot immediately so the client doesn't wait
	snapCtx, snapCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer snapCancel()

	rawVotes, _ := db.RedisClient.HGetAll(snapCtx, "poll:"+pollID+":votes").Result()
	if snapshot, err := json.Marshal(rawVotes); err == nil {
		fmt.Fprintf(c.Writer, "event: snapshot\ndata: %s\n\n", snapshot)
		flusher.Flush()
	}

	// 5. Subscribe to the Redis Pub/Sub channel for this poll
	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()

	pubsub := db.RedisClient.Subscribe(streamCtx, "poll:"+pollID+":live")
	defer pubsub.Close()

	msgChan := pubsub.Channel()
	clientGone := c.Request.Context().Done()

	// 6. Event loop — runs until client disconnects or server shuts down
	for {
		select {
		case <-clientGone:
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "event: vote\ndata: %s\n\n", msg.Payload)
			flusher.Flush()
		}
	}
}
