package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/response"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string, userRepo domain.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			response.FailUnauthorized(c, "missing token")
			c.Abort()
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			func(*jwt.Token) (any, error) { return []byte(secret), nil },
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		)
		if err != nil || !token.Valid {
			response.FailUnauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			response.FailUnauthorized(c, "invalid user identity")
			c.Abort()
			return
		}
		user, err := userRepo.GetByID(c.Request.Context(), userID)
		if err != nil {
			response.FailUnauthorized(c, "user not found")
			c.Abort()
			return
		}
		roleID := ""
		if user.RoleID != nil {
			roleID = user.RoleID.String()
		}
		c.Set("user_id", user.ID.String())
		c.Set("username", user.Username)
		c.Set("is_super", user.IsSuper)
		c.Set("role_id", roleID)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return c.Query("token")
}
