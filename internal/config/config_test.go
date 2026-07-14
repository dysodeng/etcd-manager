package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAppliesServerTimeoutDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.ReadHeaderTimeout != 5*time.Second ||
		cfg.Server.ReadTimeout != 15*time.Second ||
		cfg.Server.WriteTimeout != 30*time.Second ||
		cfg.Server.IdleTimeout != 60*time.Second ||
		cfg.Server.ShutdownTimeout != 15*time.Second {
		t.Fatalf("server config = %+v", cfg.Server)
	}
}

func TestLoadParsesExplicitServerTimeouts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	contents := []byte("server:\n  port: 8080\n  read_header_timeout: 1s\n  read_timeout: 2s\n  write_timeout: 3s\n  idle_timeout: 4s\n  shutdown_timeout: 5s\n")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.ReadHeaderTimeout != time.Second ||
		cfg.Server.ReadTimeout != 2*time.Second ||
		cfg.Server.WriteTimeout != 3*time.Second ||
		cfg.Server.IdleTimeout != 4*time.Second ||
		cfg.Server.ShutdownTimeout != 5*time.Second {
		t.Fatalf("server config = %+v", cfg.Server)
	}
}
