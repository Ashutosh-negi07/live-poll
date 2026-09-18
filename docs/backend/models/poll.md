# Poll Model (`backend/models/poll.go`)

## What it does
Defines all Go types related to polls — the database document shape, embedded options, input validation structs, the live result shape (merged MongoDB + Redis data), and the vote audit log.

## Type Overview

| Type | Stored in | Purpose |
|:---|:---|:---|
| `Option` | Embedded in `Poll` | One selectable answer inside a poll |
| `Poll` | MongoDB `polls` | Core poll document |
| `CreatePollInput` | HTTP request body | Validated input for creating a poll |
| `VoteInput` | HTTP request body | Which option the audience member chose |
| `PollResult` | API response only | Poll + live Redis counts merged together |
| `VoteLog` | MongoDB `vote_logs` | Permanent audit record of each individual vote |

## Embedded Documents (MongoDB)

```go
type Poll struct {
    Options []Option `bson:"options" json:"options"`
}
```

MongoDB stores the options array directly inside the poll document — no separate table or join needed. This is why a document database fits naturally here: polls don't have a fixed number of options.

```json
{
  "_id": "65f1...",
  "title": "Favourite framework?",
  "options": [
    { "id": "opt_0", "text": "React" },
    { "id": "opt_1", "text": "Vue" }
  ]
}
```

## `dive` Validation Tag

```go
Options []string `binding:"required,min=2,max=10,dive,required,min=1,max=100"`
```

- `min=2,max=10` — the slice must have between 2 and 10 elements
- `dive` — instructs the validator to descend into the slice and validate each string element individually
- `required,min=1,max=100` — each option string must be non-empty and under 100 characters

## `PollResult` — Struct Embedding

```go
type PollResult struct {
    Poll                        // embeds all Poll fields directly
    Votes      map[string]int64 `json:"votes"`
    TotalVotes int64            `json:"total_votes"`
}
```

Go struct embedding works like inheritance — `PollResult` automatically has all the fields of `Poll` plus the extra Redis fields. When serialized to JSON, all fields appear at the top level:

```json
{
  "id": "65f1...",
  "title": "...",
  "options": [...],
  "votes": { "opt_0": 42, "opt_1": 17 },
  "total_votes": 59
}
```

## `VoteLog` — Why Store This in MongoDB?

Redis `SADD` only stores a hash to prevent double voting — it doesn't record *when* the vote happened or which option was chosen. `VoteLog` in MongoDB provides:
1. **Audit trail** — dispute resolution if vote counts are questioned
2. **Analytics** — voting patterns over time
3. **Recovery** — if Redis data is ever lost, vote logs can be used to reconstruct tallies

## The Code

```go
package models

import (
    "time"
    "go.mongodb.org/mongo-driver/v2/bson"
)

type Option struct {
    ID   string `bson:"id"   json:"id"`
    Text string `bson:"text" json:"text"`
}

type Poll struct {
    ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Title       string        `bson:"title"         json:"title"`
    Description string        `bson:"description"   json:"description"`
    Options     []Option      `bson:"options"       json:"options"`
    CreatorID   bson.ObjectID `bson:"creator_id"    json:"creator_id"`
    CreatorName string        `bson:"creator_name"  json:"creator_name"`
    IsActive    bool          `bson:"is_active"     json:"is_active"`
    CreatedAt   time.Time     `bson:"created_at"    json:"created_at"`
}

type CreatePollInput struct {
    Title       string   `json:"title"       binding:"required,min=3,max=200"`
    Description string   `json:"description" binding:"max=500"`
    Options     []string `json:"options"     binding:"required,min=2,max=10,dive,required,min=1,max=100"`
}

type VoteInput struct {
    OptionID string `json:"option_id" binding:"required"`
}

type PollResult struct {
    Poll
    Votes      map[string]int64 `json:"votes"`
    TotalVotes int64            `json:"total_votes"`
}

type VoteLog struct {
    ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
    PollID    bson.ObjectID `bson:"poll_id"       json:"poll_id"`
    OptionID  string        `bson:"option_id"     json:"option_id"`
    VoterHash string        `bson:"voter_hash"    json:"voter_hash"`
    CreatedAt time.Time     `bson:"created_at"    json:"created_at"`
}
```
