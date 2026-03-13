package pgsql

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;not null;default:uuid_generate_v7();primaryKey"`
	Username     string    `gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string    `gorm:"size:255;not null"`
	Role         string    `gorm:"size:16;not null;default:viewer"`
	CreatedAt    time.Time `gorm:"type:timestamp(0) without time zone;not null;index"`
	UpdatedAt    time.Time `gorm:"type:timestamp(0) without time zone;not null"`
}

func (User) TableName() string { return "users" }

type Environment struct {
	ID          uuid.UUID `gorm:"type:uuid;not null;default:uuid_generate_v7();primaryKey"`
	Name        string    `gorm:"uniqueIndex;size:64;not null"`
	KeyPrefix   string    `gorm:"size:255;not null"`
	Description string    `gorm:"size:255"`
	SortOrder   int       `gorm:"default:0"`
	CreatedAt   time.Time `gorm:"type:timestamp(0) without time zone;not null;index"`
	UpdatedAt   time.Time `gorm:"type:timestamp(0) without time zone;not null"`
}

func (Environment) TableName() string { return "environments" }

type ConfigRevision struct {
	ID            uuid.UUID `gorm:"type:uuid;not null;default:uuid_generate_v7();primaryKey"`
	EnvironmentID uuid.UUID `gorm:"type:uuid;index;not null"`
	Key           string    `gorm:"size:512;not null;index"`
	Value         string    `gorm:"type:text"`
	PrevValue     string    `gorm:"type:text"`
	EtcdRevision  int64
	Action        string    `gorm:"size:16;not null"`
	Operator      uuid.UUID `gorm:"type:uuid"`
	Comment       string    `gorm:"size:512"`
	CreatedAt     time.Time `gorm:"type:timestamp(0) without time zone;not null;index"`
	UpdatedAt     time.Time `gorm:"type:timestamp(0) without time zone;not null"`
}

func (ConfigRevision) TableName() string { return "config_revisions" }

type AuditLog struct {
	ID           uuid.UUID `gorm:"type:uuid;not null;default:uuid_generate_v7();primaryKey"`
	UserID       uuid.UUID `gorm:"type:uuid;index"`
	Action       string    `gorm:"size:64;not null;index"`
	ResourceType string    `gorm:"size:64;not null"`
	ResourceKey  string    `gorm:"size:512"`
	Detail       string    `gorm:"type:text"`
	IP           string    `gorm:"size:45"`
	CreatedAt    time.Time `gorm:"type:timestamp(0) without time zone;not null;index"`
	UpdatedAt    time.Time `gorm:"type:timestamp(0) without time zone;not null"`
}

func (AuditLog) TableName() string { return "audit_logs" }
