package model

import "time"

type ConfigRevision struct {
	ID            string    `gorm:"primaryKey;size:36" json:"id"`
	EnvironmentID string    `gorm:"size:36;index;not null" json:"environment_id"`
	Key           string    `gorm:"size:512;not null;index" json:"key"`
	Value         string    `gorm:"type:text" json:"value"`
	PrevValue     string    `gorm:"type:text" json:"prev_value"`
	EtcdRevision  int64     `json:"etcd_revision"`
	Action        string    `gorm:"size:16;not null" json:"action"`
	Operator      string    `gorm:"size:36" json:"operator"`
	Comment       string    `gorm:"size:512" json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ConfigRevision) TableName() string {
	return "config_revisions"
}
