package store

import (
	"context"
	"time"

	"github.com/dysodeng/config-center/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	List(ctx context.Context, page, pageSize int) ([]model.User, int64, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uint) error
}

type EnvironmentRepository interface {
	Create(ctx context.Context, env *model.Environment) error
	GetByID(ctx context.Context, id uint) (*model.Environment, error)
	GetByName(ctx context.Context, name string) (*model.Environment, error)
	List(ctx context.Context) ([]model.Environment, error)
	Update(ctx context.Context, env *model.Environment) error
	Delete(ctx context.Context, id uint) error
}

type ConfigRevisionRepository interface {
	Create(ctx context.Context, rev *model.ConfigRevision) error
	ListByKey(ctx context.Context, envID uint, key string, page, pageSize int) ([]model.ConfigRevision, int64, error)
	GetByID(ctx context.Context, id uint) (*model.ConfigRevision, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	List(ctx context.Context, filter AuditLogFilter, page, pageSize int) ([]model.AuditLog, int64, error)
}

type AuditLogFilter struct {
	UserID       *uint
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
}
