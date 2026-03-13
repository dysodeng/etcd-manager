package pgsql

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dysodeng/config-center/internal/domain"
)

type ConfigRevisionRepository struct{ db *gorm.DB }

func NewConfigRevisionRepository(db *gorm.DB) *ConfigRevisionRepository {
	return &ConfigRevisionRepository{db: db}
}

func (r *ConfigRevisionRepository) Create(ctx context.Context, rev *domain.ConfigRevision) error {
	m := revToModel(rev)
	if err := GetDB(ctx, r.db).Create(m).Error; err != nil {
		return err
	}
	rev.ID = m.ID
	rev.CreatedAt = m.CreatedAt
	rev.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *ConfigRevisionRepository) ListByKey(ctx context.Context, envID uuid.UUID, key string, page, pageSize int) ([]domain.ConfigRevision, int64, error) {
	var models []ConfigRevision
	var total int64
	db := GetDB(ctx, r.db).Model(&ConfigRevision{}).Where("environment_id = ? AND key = ?", envID, key)
	db.Count(&total)
	if err := db.Offset((page-1)*pageSize).Limit(pageSize).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	revs := make([]domain.ConfigRevision, len(models))
	for i := range models {
		revs[i] = *revToDomain(&models[i])
	}
	return revs, total, nil
}

func (r *ConfigRevisionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ConfigRevision, error) {
	var m ConfigRevision
	if err := GetDB(ctx, r.db).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return revToDomain(&m), nil
}

func revToDomain(m *ConfigRevision) *domain.ConfigRevision {
	return &domain.ConfigRevision{
		ID:            m.ID,
		EnvironmentID: m.EnvironmentID,
		Key:           m.Key,
		Value:         m.Value,
		PrevValue:     m.PrevValue,
		EtcdRevision:  m.EtcdRevision,
		Action:        m.Action,
		Operator:      m.Operator,
		Comment:       m.Comment,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func revToModel(d *domain.ConfigRevision) *ConfigRevision {
	return &ConfigRevision{
		ID:            d.ID,
		EnvironmentID: d.EnvironmentID,
		Key:           d.Key,
		Value:         d.Value,
		PrevValue:     d.PrevValue,
		EtcdRevision:  d.EtcdRevision,
		Action:        d.Action,
		Operator:      d.Operator,
		Comment:       d.Comment,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
