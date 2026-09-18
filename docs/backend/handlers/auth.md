# Auth Handlers Documentation (`backend/handlers/auth.go`)

## 1. What does `handlers/auth.go` do?
`auth.go` contains the controller functions for user authentication:
- **`Register` (`POST /api/auth/register`)**: Validates registration data, checks for duplicate email addresses, hashes the plaintext password with bcrypt, inserts a new user record into MongoDB, and issues a JWT token.
- **`Login` (`POST /api/auth/login`)**: Finds the user in MongoDB, compares the submitted password against the stored bcrypt hash, and issues a fresh JWT token upon successful authentication.
- **`GetMe` (`GET /api/auth/me`)**: Returns the authenticated user profile using the user ID saved in the request context by the JWT middleware.

---

## 2. Why is it used?
The project specification requires that poll creation is restricted to authenticated users. Auth handlers provide the secure onboarding and session management entry points for poll creators.

---

## 3. Go Concepts & Security Explained for Beginners

### A. Bcrypt Password Hashing (`bcrypt.GenerateFromPassword`)
- Passwords must **never** be stored in plain text or with fast algorithms like MD5 or SHA-256.
- Bcrypt is an adaptive hashing function designed to be deliberately slow.
- It automatically generates a cryptographically random **salt** and blends it into the hash.
- `bcrypt.CompareHashAndPassword(hashedBytes, passwordBytes)` compares the input safely in constant time, preventing timing-attack vulnerabilities.

### B. MongoDB Document Insertion with Go Driver v2
- In Go MongoDB driver v2:
  ```go
  collection := db.GetCollection("users")
  result, err := collection.InsertOne(ctx, newUser)
  ```
- Go structs tagged with `bson:"..."` are serialized directly into binary BSON format for storage.

### C. JWT Token Generation
- Tokens are built using `jwt.NewWithClaims(jwt.SigningMethodHS256, claims)`.
- Signed into a string with `token.SignedString([]byte(cfg.JWTSecret))`.

---

## 4. Handler Breakdown & Internal Mechanics

```go
func Register(cfg *config.Config) gin.HandlerFunc
```
1. **Binding & Validation:**
   ```go
   var input models.RegisterInput
   if err := c.ShouldBindJSON(&input); err != nil {
       c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
       return
   }
   ```
2. **Duplicate Check:** Queries MongoDB `users` collection to check if `input.Email` already exists. If found, returns `409 Conflict`.
3. **Password Hashing:**
   ```go
   hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
   ```
4. **Insert into MongoDB:** Saves the user document with `CreatedAt` and `UpdatedAt` timestamps.
5. **Issue Token:** Generates a 24-hour signed JWT and responds with `201 Created`.

```go
func Login(cfg *config.Config) gin.HandlerFunc
```
1. Binds JSON payload (`email`, `password`).
2. Queries MongoDB by email:
   ```go
   err := collection.FindOne(ctx, bson.M{"email": strings.ToLower(input.Email)}).Decode(&user)
   ```
   If not found, returns `401 Unauthorized` (generic message to avoid email enumeration).
3. Compares password hash with `bcrypt.CompareHashAndPassword`.
4. Issues a signed JWT and responds with `200 OK`.
