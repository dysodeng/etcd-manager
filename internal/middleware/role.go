package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/handler"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			handler.Fail(c, handler.CodeForbidden, "admin role required")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireAdminForWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" {
			c.Next()
			return
		}
		role, _ := c.Get("role")
		if role != "admin" {
			handler.Fail(c, handler.CodeForbidden, "admin role required for write operations")
			c.Abort()
			return
		}
		c.Next()
	}
}
