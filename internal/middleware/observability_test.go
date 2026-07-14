package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDForwardsAndGeneratesID(t *testing.T) {
	tests := []struct {
		name     string
		supplied string
	}{
		{name: "forwarded", supplied: "request-from-proxy"},
		{name: "generated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			var contextID string
			router.Use(RequestID())
			router.GET("/", func(c *gin.Context) {
				contextID = RequestIDFromContext(c.Request.Context())
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set(RequestIDHeader, tt.supplied)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			got := recorder.Header().Get(RequestIDHeader)
			if got == "" || got != contextID {
				t.Fatalf("header = %q, context = %q", got, contextID)
			}
			if tt.supplied != "" && got != tt.supplied {
				t.Fatalf("request id = %q, want %q", got, tt.supplied)
			}
		})
	}
}

func TestAccessLoggerWritesStructuredFields(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := gin.New()
	router.Use(RequestID(), AccessLogger(logger), Recovery(logger))
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusCreated) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ok?token=secret", nil))

	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"request_id", "method", "path", "status", "latency_ms", "client_ip"} {
		if _, ok := event[key]; !ok {
			t.Fatalf("missing %s in %v", key, event)
		}
	}
	if strings.Contains(output.String(), "token=secret") {
		t.Fatal("query string leaked into access log")
	}
}

func TestRecoveryReturnsInternalError(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := gin.New()
	router.Use(RequestID(), Recovery(logger))
	router.GET("/panic", func(*gin.Context) { panic("boom") })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(output.String(), "request_id") || !strings.Contains(output.String(), "panic") {
		t.Fatalf("log = %s", output.String())
	}
}
