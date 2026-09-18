package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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

	count, err := db.GetCollection("polls").CountDocuments(ctx, bson.M{"_id": objID})
	if err != nil || count == 0 {
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

	// 4. Send the current vote snapshot immediately so the client doesn't wait
	snapCtx, snapCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer snapCancel()

	rawVotes, _ := db.RedisClient.HGetAll(snapCtx, "poll:"+pollID+":votes").Result()
	snapshot := buildSSEData(rawVotes)
	fmt.Fprintf(c.Writer, "event: snapshot\ndata: %s\n\n", snapshot)
	flusher.Flush()

	// 5. Subscribe to the Redis Pub/Sub channel for this poll
	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()

	pubsub := db.RedisClient.Subscribe(streamCtx, "poll:"+pollID+":live")
	defer pubsub.Close()

	msgChan := pubsub.Channel()

	// 6. Watch for client disconnect using Gin's request context
	clientGone := c.Request.Context().Done()

	// 7. Event loop — runs until client disconnects or server shuts down
	for {
		select {
		case <-clientGone:
			// Browser tab closed or network dropped — stop the goroutine
			return

		case msg, ok := <-msgChan:
			if !ok {
				// Redis channel was closed (server shutdown)
				return
			}
			// Write SSE event: the msg.Payload is already JSON from the vote handler
			fmt.Fprintf(c.Writer, "event: vote\ndata: %s\n\n", msg.Payload)
			flusher.Flush()
		}
	}
}

// buildSSEData converts a raw Redis HGetAll result (map[string]string)
// into a JSON string suitable for an SSE data field.
func buildSSEData(rawVotes map[string]string) string {
	converted := make(map[string]int64, len(rawVotes))
	for k, v := range rawVotes {
		n, _ := strconv.ParseInt(v, 10, 64)
		converted[k] = n
	}

	// Manual JSON build to avoid importing encoding/json for a simple map
	result := "{"
	first := true
	for k, v := range converted {
		if !first {
			result += ","
		}
		result += fmt.Sprintf("%q:%d", k, v)
		first = false
	}
	result += "}"
	return result
}
