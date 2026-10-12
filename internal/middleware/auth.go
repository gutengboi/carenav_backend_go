package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const UserContextKey = "user"

type claims struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// AuthenticateToken mirrors src/middleware/authenticate.ts: verifies the
// bearer JWT, loads the user (minus password) and stores it on the
// gin.Context for downstream handlers.
func AuthenticateToken(users *mongo.Collection, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		parsed := &claims{}
		_, err := jwt.ParseWithClaims(tokenStr, parsed, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			return
		}

		oid, err := bson.ObjectIDFromHex(parsed.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var user models.User
		err = users.FindOne(ctx, bson.M{"_id": oid}).Decode(&user)
		if err == mongo.ErrNoDocuments {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "User not found"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			return
		}

		c.Set(UserContextKey, &user)
		c.Next()
	}
}

// AuthorizeRole mirrors src/middleware/auth.ts.
func AuthorizeRole(allowed ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.Get(UserContextKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Access denied: insufficient role"})
			return
		}
		user := raw.(*models.User)

		for _, role := range allowed {
			if user.Role == role {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Access denied: insufficient role"})
	}
}

// CurrentUser is a small helper for handlers to pull the authenticated
// user set by AuthenticateToken.
func CurrentUser(c *gin.Context) *models.User {
	raw, ok := c.Get(UserContextKey)
	if !ok {
		return nil
	}
	return raw.(*models.User)
}
