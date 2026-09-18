package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Option is one selectable answer inside a poll.
type Option struct {
	ID   string `bson:"id"   json:"id"`   // e.g. "opt_0", "opt_1"
	Text string `bson:"text" json:"text"` // e.g. "React"
}

// Poll is the document stored in MongoDB's "polls" collection.
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

// CreatePollInput is the expected JSON body for POST /api/polls.
type CreatePollInput struct {
	Title       string   `json:"title"       binding:"required,min=3,max=200"`
	Description string   `json:"description" binding:"max=500"`
	Options     []string `json:"options"     binding:"required,min=2,max=10,dive,required,min=1,max=100"`
}

// VoteInput is the expected JSON body for POST /api/polls/:id/vote.
type VoteInput struct {
	OptionID string `json:"option_id" binding:"required"`
}

// PollResult is a poll with live vote counts merged in from Redis.
type PollResult struct {
	Poll
	Votes      map[string]int64 `json:"votes"`       // optionID → count from Redis
	TotalVotes int64            `json:"total_votes"`
}

// VoteLog is stored in MongoDB's "vote_logs" collection for audit trail.
type VoteLog struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PollID    bson.ObjectID `bson:"poll_id"       json:"poll_id"`
	OptionID  string        `bson:"option_id"     json:"option_id"`
	VoterHash string        `bson:"voter_hash"    json:"voter_hash"`
	CreatedAt time.Time     `bson:"created_at"    json:"created_at"`
}
