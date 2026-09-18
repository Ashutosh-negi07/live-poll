# User Model (`backend/models/user.go`)

## What it does
Defines all Go types related to users — the database document shape, the request input shapes, and the safe response shape sent back to the frontend.

## Why separate structs for DB vs Input vs Response?

| Struct | Used for | Why separate? |
|:---|:---|:---|
| `User` | MongoDB document | Contains `PasswordHash` — never exposed |
| `RegisterInput` | HTTP request body | Validates only the fields the user can set |
| `LoginInput` | HTTP request body | Only email + password needed |
| `AuthResponse` | HTTP response | Curated safe output — no hash, just token + profile |

If we used `User` directly for input binding, a malicious client could inject fields like `created_at` or try to set arbitrary database fields. Separate input structs prevent this.

## Struct Tags Explained

```go
type User struct {
    ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Name         string        `bson:"name"          json:"name"`
    Email        string        `bson:"email"         json:"email"`
    PasswordHash string        `bson:"password_hash" json:"-"`
    CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
}
```

| Tag | Meaning |
|:---|:---|
| `bson:"_id,omitempty"` | MongoDB stores this as `_id`. `omitempty` skips it when creating (MongoDB auto-generates it) |
| `json:"id"` | Frontend receives `"id"`, not `"ID"` or `"_id"` |
| `json:"-"` | **This field is completely excluded from all JSON output.** The password hash is never sent to any client. |
| `bson:"password_hash"` | Stored in MongoDB as `password_hash` (snake_case), not `PasswordHash` |

## Validation Tags

```go
type RegisterInput struct {
    Name     string `binding:"required,min=2,max=50"`
    Email    string `binding:"required,email"`
    Password string `binding:"required,min=6"`
}
```

Gin's validator runs these checks before the handler body executes:
- `required` — field must be present and non-empty
- `email` — must be a valid email address format
- `min=6` — string must be at least 6 characters
- `max=50` — string cannot exceed 50 characters

If any check fails, Gin automatically returns `400 Bad Request` with a validation error — no handler code needed.

## `bson.ObjectID`
MongoDB's primary key type. 12 bytes: 4-byte timestamp + 5-byte random + 3-byte counter. Globally unique without coordination. In JSON responses it serializes to a 24-character hex string: `"65f1a2b3c4d5e6f7a8b9c0d1"`.

## The Code

```go
package models

import (
    "time"
    "go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
    ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Name         string        `bson:"name"          json:"name"`
    Email        string        `bson:"email"         json:"email"`
    PasswordHash string        `bson:"password_hash" json:"-"`
    CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
}

type RegisterInput struct {
    Name     string `json:"name"     binding:"required,min=2,max=50"`
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
}

type LoginInput struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
    Token string `json:"token"`
    User  struct {
        ID    string `json:"id"`
        Name  string `json:"name"`
        Email string `json:"email"`
    } `json:"user"`
}
```
