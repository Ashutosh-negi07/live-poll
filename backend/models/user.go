package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User is the document stored in MongoDB's "users" collection.
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string        `bson:"name"          json:"name"`
	Email        string        `bson:"email"         json:"email"`
	PasswordHash string        `bson:"password_hash" json:"-"` // never sent to frontend
	CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
}

// RegisterInput is the expected JSON body for POST /api/auth/register.
type RegisterInput struct {
	Name     string `json:"name"     binding:"required,min=2,max=50"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginInput is the expected JSON body for POST /api/auth/login.
type LoginInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is what we send back after a successful register or login.
type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}
