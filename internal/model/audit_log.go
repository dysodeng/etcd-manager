package model

import "time"

type AuditLog struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	UserID       string    `gorm:"size:36;index" json:"user_id"`
	Action       string    `gorm:"size:64;not null;index" json:"action"`
	ResourceType string    `gorm:"size:64;not null" json:"resource_type"`
	ResourceKey  string    `gorm:"size:512" json:"resource_key"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IP           string    `gorm:"size:45" json:"ip"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
