package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestConfigServiceWritesValidateBeforeAccessingDependencies(t *testing.T) {
	svc := &ConfigService{}
	writes := map[string]func() error{
		"create": func() error {
			return svc.Create(context.Background(), "dev", "app.yaml", "items: [one, two", "test", uuid.Nil)
		},
		"update": func() error {
			return svc.Update(context.Background(), "dev", "app.yaml", "items: [one, two", "test", uuid.Nil)
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			err := write()
			var validationErr *ConfigValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("write error = %v, want *ConfigValidationError", err)
			}
		})
	}
}

func TestConfigServiceImportDryRunValidatesConfigValues(t *testing.T) {
	svc := &ConfigService{}
	data := []byte(`{"good.yaml":"name: app\\n","bad.json":"{\\\"broken\\\":"}`)
	result, err := svc.Import(context.Background(), "dev", data, true, uuid.Nil)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Total != 2 || result.Success != 1 || len(result.Failed) != 1 {
		t.Fatalf("Import() result = %+v, want total=2 success=1 failed=1", result)
	}
	if !strings.Contains(result.Failed[0], "bad.json") ||
		!strings.Contains(result.Failed[0], "invalid JSON config") {
		t.Fatalf("Import() failure = %q, want key and validation reason", result.Failed[0])
	}
}
