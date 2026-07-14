package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/response"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey{}).(string)
	return id
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if id == "" || len(id) > 128 {
			id = uuid.NewString()
		}
		c.Header(RequestIDHeader, id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDContextKey{}, id))
		c.Next()
	}
}

func AccessLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		logger.InfoContext(
			c.Request.Context(),
			"http request",
			"request_id", RequestIDFromContext(c.Request.Context()),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(
					c.Request.Context(),
					"http panic",
					"request_id", RequestIDFromContext(c.Request.Context()),
					"panic", fmt.Sprint(recovered),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
					Code:    response.CodeInternalError,
					Message: "internal server error",
				})
			}
		}()
		c.Next()
	}
}
