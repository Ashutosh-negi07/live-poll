# Poll Model Documentation (`backend/models/poll.go`)

## 1. What is `poll.go`?
`poll.go` defines the data structures representing polls, voting options, vote audit logs, and incoming API payloads for creating and voting on polls. It structures how polls are persisted in MongoDB and how live results are formatted when fetched from Redis.

---

## 2. Why is it used?
1. **Core Domain Model:** Live polling is the primary business requirement of this project. `poll.go` models the relationship between a poll creator, the poll question, its multiple options, and voting records.
2. **MongoDB Document Modeling:** Polls have nested options (`[]Option`). A document database like MongoDB naturally stores these embedded arrays without requiring multiple relational table joins.
3. **Strict Request Validation:** Enforces business rules at the API boundary:
   - A poll must have a title (minimum 3 characters).
   - A poll must have at least 2 options and no more than 10 options.
   - Each option text cannot be empty.

---

## 3. Go Concepts Explained for Beginners

### A. Nested Structs & Slices
- A **slice** (`[]Option`) is Go's dynamically-sized array.
- In Go, a struct can embed other structs. The `Poll` struct embeds a slice of `Option` structs:
  ```go
  type Option struct {
      ID    string `bson:"id" json:"id"`
      Text  string `bson:"text" json:"text"`
      Votes int64  `bson:"votes" json:"votes"`
  }
  ```

### B. Custom Validation Tags
- Gin utilizes `go-playground/validator`.
- `binding:"required,min=2,max=10,dive"`:
  - `min=2,max=10`: Ensures the array has between 2 and 10 options.
  - `dive`: Instructs the validator to inspect and validate each individual element inside the slice.

---

## 4. Struct Definitions & Field Breakdown

```go
// Option represents a selectable answer choice inside a poll.
type Option struct {
    ID    string `bson:"id" json:"id"`       // Unique option ID (e.g. "opt_1", "opt_2")
    Text  string `bson:"text" json:"text"`   // User-facing text description
    Votes int64  `bson:"votes" json:"votes"` // Current vote tally (synchronized with Redis)
}

// Poll represents a live poll document stored in MongoDB.
type Poll struct {
    ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Title       string        `bson:"title" json:"title"`
    Description string        `bson:"description,omitempty" json:"description,omitempty"`
    CreatorID   bson.ObjectID `bson:"creator_id" json:"creator_id"`
    CreatorName string        `bson:"creator_name,omitempty" json:"creator_name,omitempty"`
    Options     []Option      `bson:"options" json:"options"`
    IsActive    bool          `bson:"is_active" json:"is_active"`
    TotalVotes  int64         `bson:"total_votes" json:"total_votes"`
    CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

// CreatePollInput represents the payload submitted by authenticated creators.
type CreatePollInput struct {
    Title       string   `json:"title" binding:"required,min=3,max=200"`
    Description string   `json:"description" binding:"max=500"`
    Options     []string `json:"options" binding:"required,min=2,max=10,dive,min=1,max=100"`
}

// VoteInput represents a vote cast by an audience member.
type VoteInput struct {
    OptionID string `json:"option_id" binding:"required"`
}

// VoteLog records an individual vote to prevent double-voting and maintain audit trails.
type VoteLog struct {
    ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
    PollID     bson.ObjectID `bson:"poll_id" json:"poll_id"`
    OptionID   string        `bson:"option_id" json:"option_id"`
    VoterHash  string        `bson:"voter_hash" json:"voter_hash"` // Hashed IP or UserID
    CreatedAt  time.Time     `bson:"created_at" json:"created_at"`
}

// LivePollUpdate is the payload published to Redis and streamed via SSE to clients.
type LivePollUpdate struct {
    PollID     string           `json:"poll_id"`
    TotalVotes int64            `json:"total_votes"`
    Votes      map[string]int64 `json:"votes"`      // Map of OptionID -> Count
    UpdatedAt  int64            `json:"updated_at"` // Unix timestamp in milliseconds
}
```

---

## 5. Storage Strategy: Redis vs MongoDB
- **Redis (`poll:<id>:votes`):** Stores hot vote counts as a Hash (`HINCRBY poll:123:votes opt_1 1`).
- **Redis (`poll:<id>:voters`):** Stores voter hashes in a Set (`SADD poll:123:voters <voter_hash>`) for O(1) duplicate vote prevention.
- **MongoDB (`polls` & `vote_logs`):** Stores permanent historical records and detailed audit logs.
