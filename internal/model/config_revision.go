package model

import "time"

type ConfigRevision struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	EnvironmentID uint      `gorm:"index;not null" json:"environment_id"`
	Key           string    `gorm:"size:512;not null;index" json:"key"`
	Value         string    `gorm:"type:text" json:"value"`
	PrevValue     string    `gorm:"type:text" json:"prev_value"`
	EtcdRevision  int64     `json:"etcd_revision"`
	Action        string    `gorm:"size:16;not null" json:"action"`
	Operator      uint      `json:"operator"`
	Comment       string    `gorm:"size:512" json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}
