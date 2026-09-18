package middleware

import (
	"net/http"
	"strings"

	"github.com/Ashutosh-negi07/live-poll/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims defines the payload we embed inside every JWT we sign.
type Claims struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// AuthMiddleware returns a Gin handler that validates the JWT on every request.
// Protected routes wrap themselves with this middleware.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Read the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			return
		}

		// 2. Expect "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header format must be: Bearer <token>",
			})
			return
		}
		tokenString := parts[1]

		// 3. Parse and verify the token signature
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			// Guard against algorithm confusion attacks —
			// reject anything that isn't HMAC (e.g. "none" or RSA)
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})

		// 4. Handle invalid or expired tokens
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// 5. Attach identity to the context for downstream handlers
		c.Set("userID", claims.UserID)
		c.Set("name", claims.Name)
		c.Next()
	}
}
