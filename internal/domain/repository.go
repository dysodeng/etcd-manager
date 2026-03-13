package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	List(ctx context.Context, page, pageSize int) ([]User, int64, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type EnvironmentRepository interface {
	Create(ctx context.Context, env *Environment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Environment, error)
	GetByName(ctx context.Context, name string) (*Environment, error)
	List(ctx context.Context) ([]Environment, error)
	Update(ctx context.Context, env *Environment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ConfigRevisionRepository interface {
	Create(ctx context.Context, rev *ConfigRevision) error
	ListByKey(ctx context.Context, envID uuid.UUID, key string, page, pageSize int) ([]ConfigRevision, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ConfigRevision, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, filter AuditLogFilter, page, pageSize int) ([]AuditLog, int64, error)
}

type AuditLogFilter struct {
	UserID       *uuid.UUID
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
}
