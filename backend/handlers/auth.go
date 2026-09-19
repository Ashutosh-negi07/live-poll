package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Ashutosh-negi07/live-poll/config"
	"github.com/Ashutosh-negi07/live-poll/db"
	"github.com/Ashutosh-negi07/live-poll/middleware"
	"github.com/Ashutosh-negi07/live-poll/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

// Register handles POST /api/auth/register
func Register(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Bind and validate request body
		var input models.RegisterInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": friendlyError(err)})
			return
		}
		input.Email = strings.ToLower(strings.TrimSpace(input.Email))

		// 2. Check if email already exists
		collection := db.GetCollection("users")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var existing models.User
		err := collection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&existing)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}

		// 3. Hash the password
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process password"})
			return
		}

		// 4. Build and insert the user document
		newUser := models.User{
			ID:           bson.NewObjectID(),
			Name:         strings.TrimSpace(input.Name),
			Email:        input.Email,
			PasswordHash: string(hash),
			CreatedAt:    time.Now().UTC(),
		}

		if _, err := collection.InsertOne(ctx, newUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
			return
		}

		// 5. Issue JWT and respond
		tokenString, err := generateToken(cfg, newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
			return
		}

		resp := models.AuthResponse{Token: tokenString}
		resp.User.ID = newUser.ID.Hex()
		resp.User.Name = newUser.Name
		resp.User.Email = newUser.Email

		c.JSON(http.StatusCreated, resp)
	}
}

// Login handles POST /api/auth/login
func Login(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Bind and validate request body
		var input models.LoginInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": friendlyError(err)})
			return
		}

		input.Email = strings.ToLower(strings.TrimSpace(input.Email))

		// 2. Look up the user by email
		collection := db.GetCollection("users")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user models.User
		if err := collection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&user); err != nil {
			// Generic message — don't reveal whether the email exists
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}

		// 3. Compare submitted password with stored hash
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}

		// 4. Issue JWT
		tokenString, err := generateToken(cfg, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
			return
		}

		// 5. Respond
		resp := models.AuthResponse{Token: tokenString}
		resp.User.ID = user.ID.Hex()
		resp.User.Name = user.Name
		resp.User.Email = user.Email

		c.JSON(http.StatusOK, resp)
	}
}

// GetMe handles GET /api/auth/me — returns the logged-in user's profile.
// Requires AuthMiddleware to have run first.
func GetMe(c *gin.Context) {
	userID := c.MustGet("userID").(string)

	objID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	collection := db.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// User struct has json:"-" on PasswordHash — never included in response
	c.JSON(http.StatusOK, user)
}

// generateToken creates a signed JWT for the given user.
func generateToken(cfg *config.Config, user models.User) (string, error) {
	claims := &middleware.Claims{
		UserID: user.ID.Hex(),
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWTExpiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}
