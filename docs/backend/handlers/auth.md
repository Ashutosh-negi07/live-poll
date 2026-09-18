# Auth Handlers (`backend/handlers/auth.go`)

## What it does
Implements the three authentication endpoints:
- `POST /api/auth/register` — create a new account, returns JWT
- `POST /api/auth/login` — verify credentials, returns JWT
- `GET /api/auth/me` — return the logged-in user's profile (protected)

## Handler Pattern in Gin

Each handler that needs `cfg` (for `JWTSecret`, `JWTExpiryHours`) is a **factory function** — it takes `cfg` and returns a `gin.HandlerFunc`:

```go
func Register(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        // handler logic here — cfg is captured in the closure
    }
}
```

`GetMe` doesn't need `cfg` (it only reads from MongoDB and context), so it is a plain `gin.HandlerFunc` directly.

---

## `Register` — Step by Step

### 1. `ShouldBindJSON`
```go
var input models.RegisterInput
if err := c.ShouldBindJSON(&input); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}
```
Gin reads the request body, decodes JSON into `input`, and runs all `binding:` tag validations. If anything fails (missing field, email format wrong, password too short), `err` is non-nil and we immediately return `400 Bad Request`. No further logic runs.

### 2. Email Normalization
```go
input.Email = strings.ToLower(strings.TrimSpace(input.Email))
```
`"  User@Gmail.COM  "` → `"user@gmail.com"`. Prevents duplicate accounts differing only in case or whitespace.

### 3. Duplicate Check
```go
err := collection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&existing)
if err == nil {
    c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
    return
}
```
`FindOne` returns an error when no document is found (`mongo.ErrNoDocuments`). So `err == nil` means a document *was* found — the email exists. We return `409 Conflict`.

### 4. bcrypt Hashing
```go
hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
```
`bcrypt.DefaultCost` = 10. The function generates a random salt internally, hashes the password 2¹⁰ times, and returns a 60-character string that embeds the salt, cost, and hash. This takes ~100ms deliberately.

### 5. MongoDB Insert
```go
newUser := models.User{
    ID:           bson.NewObjectID(),
    ...
    PasswordHash: string(hash),
}
collection.InsertOne(ctx, newUser)
```
`bson.NewObjectID()` generates a unique 12-byte ID. The Go struct is automatically serialized to BSON using the `bson:"..."` tags.

### 6. `generateToken` (private helper)
```go
func generateToken(cfg *config.Config, user models.User) (string, error) {
    claims := &middleware.Claims{
        UserID: user.ID.Hex(),  // ObjectID → "65f1a2b3..."
        Name:   user.Name,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(...)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(cfg.JWTSecret))
}
```
- `user.ID.Hex()` converts the 12-byte `bson.ObjectID` to a 24-character hex string — safe to embed in JWT and URLs
- `jwt.SigningMethodHS256` = HMAC-SHA256
- `token.SignedString(secret)` produces the final `eyJ...` string

---

## `Login` — Security Notes

### Email Enumeration Prevention
Both "user not found" and "wrong password" return the identical message:
```go
c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
```
If we said "email not found" vs "wrong password", an attacker could enumerate valid email addresses by observing which message they get.

### `bcrypt.CompareHashAndPassword`
```go
bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
```
This is a **constant-time comparison** — it takes the same amount of time whether the password is completely wrong or off by one character. This prevents timing attacks where an attacker measures response time to deduce how close their guess was.

---

## `GetMe` — Reading from Context
```go
userID := c.MustGet("userID").(string)
```
`AuthMiddleware` must have run before this handler (enforced at route registration in `main.go`). `MustGet` panics if the key isn't set, but since the middleware guarantees it, this is safe. The `.(string)` is a Go type assertion — converts the `interface{}` stored in the context to a concrete `string`.

---

## Response Shape

**Register/Login success (`201`/`200`):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiJ9...",
  "user": {
    "id": "65f1a2b3c4d5e6f7a8b9c0d1",
    "name": "Ashutosh",
    "email": "ashutosh@example.com"
  }
}
```

**GetMe success (`200`):**
```json
{
  "id": "65f1a2b3c4d5e6f7a8b9c0d1",
  "name": "Ashutosh",
  "email": "ashutosh@example.com",
  "created_at": "2026-09-18T18:00:00Z"
}
```
Note: `password_hash` is absent — the `json:"-"` tag on the `User` struct silently omits it from every JSON response automatically.
