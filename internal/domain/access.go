package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrEnvironmentForbidden = errors.New("environment access denied")

type EnvironmentScope struct {
	Unrestricted bool
	AllowedIDs   []uuid.UUID
}

type environmentScopeKey struct{}

func WithEnvironmentScope(ctx context.Context, scope EnvironmentScope) context.Context {
	return context.WithValue(ctx, environmentScopeKey{}, scope)
}

func RequireEnvironmentAccess(ctx context.Context, environmentID uuid.UUID) error {
	scope, ok := ctx.Value(environmentScopeKey{}).(EnvironmentScope)
	if !ok {
		return ErrEnvironmentForbidden
	}
	if scope.Unrestricted {
		return nil
	}
	for _, id := range scope.AllowedIDs {
		if id == environmentID {
			return nil
		}
	}
	return ErrEnvironmentForbidden
}
