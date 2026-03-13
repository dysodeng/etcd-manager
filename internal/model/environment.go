package model

type Environment struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;size:64;not null" json:"name"`
	KeyPrefix   string `gorm:"size:255;not null" json:"key_prefix"`
	Description string `gorm:"size:255" json:"description"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
}
