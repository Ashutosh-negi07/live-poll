# JWT Authentication Middleware Documentation (`backend/middleware/auth.go`)

## 1. What does `auth.go` do?
`auth.go` implements HTTP request interception to guard private endpoints (such as poll creation and creator dashboard routes). It extracts the JSON Web Token (JWT) from the incoming `Authorization: Bearer <token>` header, cryptographically verifies its digital signature, checks expiration, and attaches the authenticated user's ID to Gin's context for downstream handlers to use.

---

## 2. Why is it used?
1. **Stateless Authentication:** Rather than storing server-side session cookies in memory, JWTs encode user identity directly inside the cryptographically signed token. This allows our backend to scale horizontally without session synchronization bottlenecks.
2. **Access Control:** Prevents unauthenticated users from creating spam polls or altering existing polls.
3. **Identity Propagation:** By injecting `userID` into the request context, downstream handlers (like `CreatePoll`) know exactly who authored the poll without having to re-query the database.

---

## 3. Go Concepts Explained for Beginners

### A. Middleware Pattern in Gin
- In Gin, middleware is simply a function with the signature `func(c *gin.Context)`.
- When an HTTP request arrives, it travels through an ordered chain of middleware before reaching the final handler:
  ```
  HTTP Request ➔ CORS Middleware ➔ Auth Middleware ➔ CreatePoll Handler
  ```
- If the token is invalid, middleware calls `c.AbortWithStatusJSON(...)`. This stops the pipeline immediately, preventing the request from ever reaching the handler.
- If the token is valid, middleware calls `c.Next()` to continue execution down the chain.

### B. Gin Context Storage (`c.Set` and `c.Get`)
- The `*gin.Context` object lives for the entire duration of an HTTP request.
- `c.Set("userID", claims.UserID)` stores the user ID in an internal key-value map on the context.
- Downstream handlers retrieve it via `userID, exists := c.Get("userID")`.

---

## 4. Code Breakdown & Internal Mechanics

```go
type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "Authorization header required",
            })
            return
        }

        // Expected format: "Bearer <token>"
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "Invalid Authorization format. Must be 'Bearer <token>'",
            })
            return
        }

        tokenString := parts[1]

        // Parse and validate token signature using HMAC-SHA256
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
            if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
            }
            return []byte(cfg.JWTSecret), nil
        })

        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "error": "Invalid or expired token",
            })
            return
        }

        // Extract claims and inject into Gin context
        claims, ok := token.Claims.(*Claims)
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Failed to parse token claims"})
            return
        }

        c.Set("userID", claims.UserID)
        c.Set("email", claims.Email)
        c.Next()
    }
}
```

---

## 5. Security Principles Applied
1. **Signature Verification:** Rejects any token that was forged or tampered with.
2. **Algorithm Whitelisting:** Explicitly checks `t.Method.(*jwt.SigningMethodHMAC)` to defeat "none" algorithm and key-confusion vulnerabilities.
3. **Expiration Enforcement:** `jwt.RegisteredClaims` automatically enforces the `exp` timestamp claim.
