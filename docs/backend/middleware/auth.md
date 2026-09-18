# JWT Auth Middleware (`backend/middleware/auth.go`)

## What it does
Intercepts every HTTP request on protected routes (poll creation, creator dashboard). Extracts the JWT from the `Authorization` header, verifies the cryptographic signature and expiry, then injects the user's identity into the Gin context for downstream handlers.

## How JWT Works

A JWT is three Base64URL-encoded parts joined by dots:
```
<header>.<payload>.<signature>
```

- **Header** — `{"alg":"HS256","typ":"JWT"}`
- **Payload** — our claims: `user_id`, `name`, `exp` (expiry unix timestamp)
- **Signature** — `HMAC-SHA256(header + "." + payload, JWT_SECRET)`

The signature is what makes it tamper-proof. The server re-computes the signature using `JWT_SECRET` and compares it to the one in the token. If anything in the header or payload was changed, the signatures won't match and the token is rejected.

## The `Claims` Struct

```go
type Claims struct {
    UserID string `json:"user_id"`
    Name   string `json:"name"`
    jwt.RegisteredClaims  // ← embeds ExpiresAt, IssuedAt, Issuer, etc.
}
```

`jwt.RegisteredClaims` is embedded — its fields become part of `Claims` directly. The `jwt` library automatically checks `ExpiresAt` during `ParseWithClaims` and returns an error if the token is expired.

## The Middleware Chain

Gin processes requests through an ordered chain:
```
Request → CORS → AuthMiddleware → Handler
                      ↓
               Token invalid?
               AbortWithStatusJSON(401)  ← chain stops here
                      ↓
               Token valid?
               c.Set("userID", ...)
               c.Next()  ← chain continues to handler
```

`c.AbortWithStatusJSON` sends the response AND stops the chain — the handler is never called.  
`c.Next()` passes execution to the next middleware or handler in the chain.

## Algorithm Confusion Attack Guard

```go
if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, jwt.ErrSignatureInvalid
}
```

Some JWT libraries accept `"alg": "none"` — a token with no signature — as valid. We explicitly type-assert that the algorithm is HMAC. Any other algorithm (RSA, ECDSA, none) causes an immediate rejection, regardless of what the token header claims.

## Why a Generic Error Message?

```go
c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
    "error": "invalid or expired token",
})
```

We don't distinguish between "token missing", "signature wrong", and "token expired" in the response. Specific errors help attackers understand exactly what they need to fix. A single generic message gives them nothing useful.

## How handlers use the injected context

```go
// Inside any protected handler:
userID := c.MustGet("userID").(string)
name   := c.MustGet("name").(string)
```

`MustGet` panics if the key isn't set — but because the middleware ran and verified the token first, it is guaranteed to be there. Gin's built-in `Recovery` middleware catches any panics and returns a 500 rather than crashing the server.

## Applying it to routes in `main.go`

```go
// Public routes — no middleware
router.POST("/api/auth/register", handlers.Register(cfg))
router.POST("/api/auth/login",    handlers.Login(cfg))

// Protected routes — auth required
auth := router.Group("/api")
auth.Use(middleware.AuthMiddleware(cfg))
{
    auth.POST("/polls",              handlers.CreatePoll(cfg))
    auth.GET("/polls/my",            handlers.ListMyPolls)
    auth.PATCH("/polls/:id/close",   handlers.ClosePoll)
    auth.GET("/auth/me",             handlers.GetMe)
}
```
