# Poll Handlers Documentation (`backend/handlers/poll.go`)

## 1. What does `handlers/poll.go` do?
`poll.go` handles CRUD operations for polls:
- **`CreatePoll` (`POST /api/polls`)**: Authenticated endpoint. Creates a new poll in MongoDB, assigns unique identifiers to each option, and pre-populates Redis hash counters with zero.
- **`GetPoll` (`GET /api/polls/:id`)**: Public endpoint. Retrieves poll metadata (question, options) from MongoDB, merges the latest live vote counts from Redis, and returns the unified state.
- **`ListMyPolls` (`GET /api/polls/my`)**: Authenticated endpoint. Returns all polls created by the logged-in user for their creator dashboard.
- **`ClosePoll` (`PATCH /api/polls/:id/close`)**: Authenticated endpoint. Allows the creator to deactivate a poll to prevent further votes.

---

## 2. Why is it used?
Polls are the centerpiece of the live polling service. Combining MongoDB for document storage with Redis for state retrieval provides high read performance: poll definitions are persistent and queryable, while live counts are always read from blazing-fast RAM.

---

## 3. Go Concepts & Database Patterns Explained

### A. Route Parameters in Gin (`c.Param("id")`)
- Endpoints with dynamic URL segments (e.g. `/api/polls/:id`) access the variable segment using `c.Param("id")`.
- We parse this string into a MongoDB `bson.ObjectIDFromHex(idStr)` to ensure valid format before querying the database.

### B. Dual-Store Synchronization (MongoDB + Redis)
- When a poll is created:
  1. Insert poll into MongoDB: Generates permanent `_id`.
  2. Initialize Redis hash:
     ```go
     // Key: "poll:<poll_id>:votes", Field: "opt_1", Value: 0
     db.RedisClient.HSet(ctx, "poll:"+pollID+":votes", "opt_1", 0, "opt_2", 0)
     ```
- When a poll is fetched:
  1. Read poll metadata from MongoDB.
  2. Read current counts from Redis with `HGetAll(ctx, "poll:"+pollID+":votes")`.
  3. Overlay Redis counts onto the options array before sending the JSON response to the client.
- **Why this design?** If Redis restarts or is cleared, MongoDB retains the permanent baseline; during live voting, Redis takes 100% of the write load off MongoDB.

---

## 4. Handler Breakdown & Internal Mechanics

```go
func CreatePoll(c *gin.Context)
```
1. Retrieves `userID` from Gin context (set by `AuthMiddleware`).
2. Validates incoming payload: `models.CreatePollInput`.
3. Transforms string option labels into `models.Option` objects with generated IDs (`opt_1`, `opt_2`, etc.).
4. Inserts the document into MongoDB `polls` collection.
5. Seeds Redis hash keys `poll:<id>:votes` with initial `0` counts.
6. Returns `201 Created` with poll shareable link metadata.

```go
func GetPoll(c *gin.Context)
```
1. Parses URL parameter `id`.
2. Queries MongoDB `polls` collection by `_id`.
3. Fetches live counts from Redis `HGetAll`.
4. Overwrites option counts with Redis values for real-time accuracy.
5. Returns `200 OK`.
