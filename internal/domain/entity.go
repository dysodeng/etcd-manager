package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	IsSuper      bool       `json:"is_super"`
	RoleID       *uuid.UUID `json:"role_id"`
	RoleName     string     `json:"role_name" gorm:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Role struct {
	ID           uuid.UUID        `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Permissions  []RolePermission `json:"permissions" gorm:"-"`
	Environments []uuid.UUID      `json:"environment_ids" gorm:"-"`
	UserCount    int64            `json:"user_count" gorm:"-"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type RolePermission struct {
	ID       uuid.UUID `json:"id"`
	RoleID   uuid.UUID `json:"role_id"`
	Module   string    `json:"module"`
	CanRead  bool      `json:"can_read"`
	CanWrite bool      `json:"can_write"`
}

type RoleEnvironment struct {
	ID            uuid.UUID `json:"id"`
	RoleID        uuid.UUID `json:"role_id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
}

type Environment struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	KeyPrefix     string    `json:"key_prefix"`
	ConfigPrefix  string    `json:"config_prefix"`
	GatewayPrefix string    `json:"gateway_prefix"`
	GrpcPrefix    string    `json:"grpc_prefix"`
	Description   string    `json:"description"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ConfigRevision struct {
	ID            uuid.UUID `json:"id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	PrevValue     string    `json:"prev_value"`
	EtcdRevision  int64     `json:"etcd_revision"`
	Action        string    `json:"action"`
	Operator      uuid.UUID `json:"operator"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceKey  string    `json:"resource_key"`
	Detail       string    `json:"detail"`
	IP           string    `json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
