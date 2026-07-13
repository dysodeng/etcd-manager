package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestRequireEnvironmentAccess(t *testing.T) {
	allowedID := uuid.New()
	deniedID := uuid.New()
	tests := []struct {
		name    string
		ctx     context.Context
		envID   uuid.UUID
		wantErr bool
	}{
		{name: "missing scope defaults to deny", ctx: context.Background(), envID: allowedID, wantErr: true},
		{name: "allowed environment", ctx: WithEnvironmentScope(context.Background(), EnvironmentScope{AllowedIDs: []uuid.UUID{allowedID}}), envID: allowedID},
		{name: "other environment denied", ctx: WithEnvironmentScope(context.Background(), EnvironmentScope{AllowedIDs: []uuid.UUID{allowedID}}), envID: deniedID, wantErr: true},
		{name: "unrestricted", ctx: WithEnvironmentScope(context.Background(), EnvironmentScope{Unrestricted: true}), envID: deniedID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireEnvironmentAccess(tt.ctx, tt.envID)
			if tt.wantErr && !errors.Is(err, ErrEnvironmentForbidden) {
				t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("error = %v, want nil", err)
			}
		})
	}
}
