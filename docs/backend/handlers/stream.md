# Real-Time SSE Stream Handler Documentation (`backend/handlers/stream.go`)

## 1. What does `handlers/stream.go` do?
`stream.go` provides the Server-Sent Events (SSE) streaming endpoint (`GET /api/polls/:id/stream`). It maintains a persistent, one-way HTTP connection between the Go server and every audience browser watching the poll. When a vote is cast, it receives the event from Redis Pub/Sub and immediately flushes the update down the open HTTP stream with **zero page refresh**.

---

## 2. Why SSE over WebSockets?
| Feature | Server-Sent Events (SSE) | WebSockets | Why SSE is the right choice here |
| :--- | :--- | :--- | :--- |
| **Protocol** | Standard HTTP/1.1 or HTTP/2 | Custom WS protocol (`ws://`) | SSE traverses firewalls, corporate proxies, and CDNs without special proxy rules. |
| **Direction** | One-way (Server ➔ Client) | Bidirectional (Full duplex) | In our app, clients send votes via standard HTTP `POST` requests; results only flow from server to client. Bidirectional communication is unnecessary overhead. |
| **Reconnection** | Built-in native browser auto-reconnect | Requires manual JS retry loops | If mobile signal flickers, the browser's `EventSource` automatically reconnects and resumes streaming. |
| **Complexity** | Simple text stream (`data: ...\n\n`) | Requires custom framing and heartbeat protocols | Radically simpler to inspect, test with `curl`, and maintain in Go. |

---

## 3. Go Concepts Explained for Beginners

### A. HTTP Streaming with `http.Flusher`
- By default, HTTP servers buffer response data and send it in large chunks when the handler returns.
- For streaming, we need the response to send immediately.
- We cast Gin's response writer to `http.Flusher`:
  ```go
  flusher, ok := c.Writer.(http.Flusher)
  ```
- Calling `flusher.Flush()` forces the Go runtime to transmit the buffered bytes over the TCP socket immediately to the browser.

### B. Redis Pub/Sub in Go (`Subscribe`)
- Each connected viewer subscribes to a Redis channel named `poll:<poll_id>:live`:
  ```go
  pubsub := db.RedisClient.Subscribe(c.Request.Context(), "poll:"+pollID+":live")
  defer pubsub.Close()
  ch := pubsub.Channel()
  ```
- Multiple viewers watching the same poll all listen to the same Redis channel without polling or querying the database.

### C. The `select` Multiplexing Loop
- In Go, a `select` statement lets a goroutine wait on multiple channel operations.
- We monitor:
  1. `msg, ok := <-ch`: A new vote update arrived from Redis! Send it to the client.
  2. `<-c.Request.Context().Done()`: The user closed their browser tab! Break the loop, unsubscribe from Redis, and exit the goroutine.
  3. `<-heartbeat.C`: Every 20 seconds, send an SSE comment (`: keepalive\n\n`) to prevent cloud load balancers (like AWS or Cloudflare) from terminating idle connections.

---

## 4. Code Breakdown & Internal Mechanics

```go
func StreamPollUpdates(c *gin.Context) {
    pollID := c.Param("id")

    // 1. Set required SSE HTTP response headers
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.Header().Set("Transfer-Encoding", "chunked")

    flusher, ok := c.Writer.(http.Flusher)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
        return
    }

    // 2. Send initial state immediately upon connection
    initialVotes, _ := db.RedisClient.HGetAll(c.Request.Context(), "poll:"+pollID+":votes").Result()
    initialPayload, _ := json.Marshal(initialVotes)
    fmt.Fprintf(c.Writer, "event: initial\ndata: %s\n\n", initialPayload)
    flusher.Flush()

    // 3. Subscribe to Redis Pub/Sub topic
    pubsub := db.RedisClient.Subscribe(c.Request.Context(), "poll:"+pollID+":live")
    defer pubsub.Close()
    ch := pubsub.Channel()

    // 4. Heartbeat ticker (keeps connection alive across proxies)
    ticker := time.NewTicker(20 * time.Second)
    defer ticker.Stop()

    // 5. Event loop
    for {
        select {
        case <-c.Request.Context().Done():
            // Client disconnected (closed tab)
            return

        case msg, ok := <-ch:
            if !ok {
                return
            }
            // Push update to client
            fmt.Fprintf(c.Writer, "event: vote\ndata: %s\n\n", msg.Payload)
            flusher.Flush()

        case <-ticker.C:
            // Send SSE heartbeat comment
            fmt.Fprintf(c.Writer, ": keepalive\n\n")
            flusher.Flush()
        }
    }
}
```

---

## 5. Testing with `curl`
You can verify the live stream directly from your terminal:
```bash
curl -N -H "Accept: text/event-stream" http://localhost:8080/api/polls/65f1234abcd/stream
```
Every time a vote is cast in another terminal, the JSON payload appears immediately in the curl output!
