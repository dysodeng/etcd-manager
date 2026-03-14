package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/response"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			response.Fail(c, response.CodeForbidden, "admin role required")
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
			response.Fail(c, response.CodeForbidden, "admin role required for write operations")
			c.Abort()
			return
		}
		c.Next()
	}
}
