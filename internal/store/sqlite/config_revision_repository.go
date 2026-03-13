package sqlite

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"gorm.io/gorm"
)

type ConfigRevisionRepository struct{ db *gorm.DB }

func NewConfigRevisionRepository(db *gorm.DB) *ConfigRevisionRepository {
	return &ConfigRevisionRepository{db: db}
}

func (r *ConfigRevisionRepository) Create(ctx context.Context, rev *model.ConfigRevision) error {
	return GetDB(ctx, r.db).Create(rev).Error
}
func (r *ConfigRevisionRepository) ListByKey(ctx context.Context, envID uint, key string, page, pageSize int) ([]model.ConfigRevision, int64, error) {
	var revs []model.ConfigRevision
	var total int64
	db := GetDB(ctx, r.db).Model(&model.ConfigRevision{}).Where("environment_id = ? AND key = ?", envID, key)
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&revs).Error
	return revs, total, err
}
func (r *ConfigRevisionRepository) GetByID(ctx context.Context, id uint) (*model.ConfigRevision, error) {
	var rev model.ConfigRevision
	return &rev, GetDB(ctx, r.db).First(&rev, id).Error
}
