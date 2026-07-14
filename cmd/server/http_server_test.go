package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dysodeng/etcd-manager/internal/config"
)

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (r *deadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	r.deadlines = append(r.deadlines, deadline)
	return nil
}

func TestResponseWriteDeadlineExemptsWatch(t *testing.T) {
	tests := []struct {
		path      string
		wantCalls int
	}{
		{path: "/api/v1/configs", wantCalls: 2},
		{path: "/api/v1/watch", wantCalls: 0},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			handler := withResponseWriteDeadline(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}),
				30*time.Second,
			)

			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if len(recorder.deadlines) != tt.wantCalls {
				t.Fatalf("%s deadline calls = %d, want %d", tt.path, len(recorder.deadlines), tt.wantCalls)
			}
		})
	}
}

func TestNewHTTPServerUsesConfiguredTimeouts(t *testing.T) {
	cfg := config.ServerConfig{
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       4 * time.Second,
	}

	server := newHTTPServer(":0", http.NotFoundHandler(), cfg)

	if server.ReadHeaderTimeout != time.Second ||
		server.ReadTimeout != 2*time.Second ||
		server.IdleTimeout != 4*time.Second ||
		server.WriteTimeout != 0 {
		t.Fatalf("server = %+v", server)
	}
}

func TestServeHTTPShutsDownWhenContextIsCanceled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	server := newHTTPServer(
		listener.Addr().String(),
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		config.ServerConfig{},
	)
	done := make(chan error, 1)
	go func() {
		done <- serveHTTP(ctx, server, listener, time.Second)
	}()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveHTTP() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveHTTP did not stop after context cancellation")
	}
}
