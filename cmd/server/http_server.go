package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/dysodeng/etcd-manager/internal/config"
)

func withResponseWriteDeadline(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if timeout > 0 && request.URL.Path != "/api/v1/watch" {
			controller := http.NewResponseController(w)
			if err := controller.SetWriteDeadline(time.Now().Add(timeout)); err == nil {
				defer controller.SetWriteDeadline(time.Time{})
			}
		}
		next.ServeHTTP(w, request)
	})
}

func newHTTPServer(address string, handler http.Handler, cfg config.ServerConfig) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           withResponseWriteDeadline(handler, cfg.WriteTimeout),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      0,
		IdleTimeout:       cfg.IdleTimeout,
	}
}

func serveHTTP(ctx context.Context, server *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	result := make(chan error, 1)
	go func() {
		result <- server.Serve(listener)
	}()

	select {
	case err := <-result:
		return normalizeServeError(err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		shutdownErr := server.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			_ = server.Close()
		}
		_ = listener.Close()
		serveErr := normalizeServeError(<-result)
		if shutdownErr != nil {
			return shutdownErr
		}
		return serveErr
	}
}

func normalizeServeError(err error) error {
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}
