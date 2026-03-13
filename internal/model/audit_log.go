package model

import "time"

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	Action       string    `gorm:"size:64;not null;index" json:"action"`
	ResourceType string    `gorm:"size:64;not null" json:"resource_type"`
	ResourceKey  string    `gorm:"size:512" json:"resource_key"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IP           string    `gorm:"size:45" json:"ip"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}
