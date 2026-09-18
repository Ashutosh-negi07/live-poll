# User Model Documentation (`backend/models/user.go`)

## 1. What is `user.go`?
`user.go` defines the data models and data structures for users within the application. It represents how user records are stored in MongoDB, how they are parsed from incoming HTTP JSON payloads, and how user profiles are safely returned without leaking sensitive password hashes.

---

## 2. Why is it used?
1. **Schema & Contract Definition:** Go is statically typed. Struct definitions act as strict contracts between our Go code, MongoDB documents, and frontend HTTP payloads.
2. **Security & Field Masking:** Password hashes must NEVER be sent back in API responses. Struct tags allow us to hide (`json:"-"`) or omit sensitive credentials.
3. **Validation at the Boundary:** Before creating a user in MongoDB, Gin's validator checks fields like `binding:"required,email"` and `binding:"required,min=6"` server-side.

---

## 3. Go Concepts Explained for Beginners

### A. Struct Tags (`json:"..."` and `bson:"..."`)
- In Go, struct fields can have metadata tags enclosed in backticks.
- `json:"email"`: Tells the JSON encoder/decoder how to serialize or deserialize this field when receiving or sending HTTP JSON.
- `bson:"_id,omitempty"`: Tells the MongoDB Go driver how to map this struct field to a BSON (Binary JSON) document in the database.
- `json:"-"`: Completely excludes the field from JSON serialization (essential for password hashes!).
- `binding:"required,email"`: Gin's validation tag. If the request body is missing this field or it's not a valid email, Gin rejects the request with HTTP 400 Bad Request.

### B. MongoDB ObjectIDs (`bson.ObjectID`)
- MongoDB uses 12-byte unique identifiers called `ObjectID` as the primary key (`_id`).
- In Go MongoDB driver v2, this is represented by `bson.ObjectID`.

---

## 4. Struct Definitions & Field Breakdown

```go
// User represents a user account stored in MongoDB.
type User struct {
    ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Email        string        `bson:"email" json:"email"`
    PasswordHash string        `bson:"password_hash" json:"-"` // Never exposed in JSON responses
    Name         string        `bson:"name" json:"name"`
    CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
    UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// RegisterInput represents the incoming payload when a user signs up.
type RegisterInput struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
    Name     string `json:"name" binding:"required,min=2"`
}

// LoginInput represents the incoming payload when a user logs in.
type LoginInput struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

// UserResponse represents safe user data returned to the frontend.
type UserResponse struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
    Token string `json:"token,omitempty"`
}
```

---

## 5. Security Principles Applied
1. **Never Store Plaintext Passwords:** Passwords are never saved as raw text. The application uses `bcrypt` with a cost factor (e.g., 12) before writing to MongoDB.
2. **Separation of Input vs Persistence Models:** Notice that `RegisterInput` and `User` are separate structs. Clients cannot inject arbitrary internal database fields (like `role` or `is_admin`) during registration.
