package service

import (
	"errors"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{name: "valid yaml", key: "app.yaml", value: "server:\n  port: 8080\n"},
		{name: "valid uppercase yml", key: "app.YML", value: "enabled: true\n"},
		{name: "invalid yaml", key: "app.yaml", value: "server: {host: localhost", wantErr: true},
		{name: "valid multi-document yaml", key: "app.yml", value: "name: first\n---\nname: second\n"},
		{name: "invalid later yaml document", key: "app.yml", value: "name: first\n---\nitems: [one, two", wantErr: true},
		{name: "valid json", key: "app.json", value: `{"server":{"port":8080}}`},
		{name: "invalid incomplete json", key: "app.json", value: `{"server":`, wantErr: true},
		{name: "invalid json trailing content", key: "app.json", value: `{"ok":true} garbage`, wantErr: true},
		{name: "valid toml", key: "app.toml", value: "[server]\nport = 8080\n"},
		{name: "invalid toml", key: "app.toml", value: "name = \"broken", wantErr: true},
		{name: "unknown suffix", key: "app.conf", value: "{ incomplete"},
		{name: "no suffix", key: "app/config", value: "{ incomplete"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.key, tt.value)
			if tt.wantErr && err == nil {
				t.Fatal("ValidateConfig() error = nil, want validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateConfig() error = %v, want nil", err)
			}
			if tt.wantErr {
				var validationErr *ConfigValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("error type = %T, want *ConfigValidationError", err)
				}
			}
		})
	}
}
