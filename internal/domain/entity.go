package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
