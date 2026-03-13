package model

import "time"

type Environment struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	KeyPrefix   string    `gorm:"size:255;not null" json:"key_prefix"`
	Description string    `gorm:"size:255" json:"description"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Environment) TableName() string {
	return "environments"
}
