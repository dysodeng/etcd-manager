package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		want  slog.Level
		valid bool
	}{
		{name: "debug", want: slog.LevelDebug, valid: true},
		{name: "INFO", want: slog.LevelInfo, valid: true},
		{name: "warn", want: slog.LevelWarn, valid: true},
		{name: "error", want: slog.LevelError, valid: true},
		{name: "verbose", want: slog.LevelInfo, valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := ParseLevel(tt.name)
			if got != tt.want || valid != tt.valid {
				t.Fatalf("ParseLevel(%q) = (%v, %v), want (%v, %v)", tt.name, got, valid, tt.want, tt.valid)
			}
		})
	}
}

func TestNewJSONLoggerHonorsLevel(t *testing.T) {
	var output bytes.Buffer
	logger, valid := NewJSONLogger(&output, "warn")
	if !valid {
		t.Fatal("warn should be a valid log level")
	}

	logger.Info("hidden")
	logger.Warn("visible", "request_id", "req-1")

	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event["msg"] != "visible" || event["request_id"] != "req-1" {
		t.Fatalf("event = %v", event)
	}
}
