package sqlite

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dysodeng/config-center/internal/domain"
	"github.com/dysodeng/config-center/internal/model"
)

type AuditLogRepository struct{ db *gorm.DB }

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	log.ID = uuid.Must(uuid.NewV7())
	return GetDB(ctx, r.db).Create(auditToModel(log)).Error
}

func (r *AuditLogRepository) List(ctx context.Context, filter domain.AuditLogFilter, page, pageSize int) ([]domain.AuditLog, int64, error) {
	var models []model.AuditLog
	var total int64
	db := GetDB(ctx, r.db).Model(&model.AuditLog{})
	if filter.UserID != nil {
		db = db.Where("user_id = ?", filter.UserID.String())
	}
	if filter.Action != "" {
		db = db.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		db = db.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.StartTime != nil {
		db = db.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		db = db.Where("created_at <= ?", *filter.EndTime)
	}
	db.Count(&total)
	if err := db.Offset((page-1)*pageSize).Limit(pageSize).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	logs := make([]domain.AuditLog, len(models))
	for i := range models {
		logs[i] = *auditToDomain(&models[i])
	}
	return logs, total, nil
}

func auditToDomain(m *model.AuditLog) *domain.AuditLog {
	return &domain.AuditLog{
		ID:           uuid.MustParse(m.ID),
		UserID:       uuid.MustParse(m.UserID),
		Action:       m.Action,
		ResourceType: m.ResourceType,
		ResourceKey:  m.ResourceKey,
		Detail:       m.Detail,
		IP:           m.IP,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func auditToModel(d *domain.AuditLog) *model.AuditLog {
	return &model.AuditLog{
		ID:           d.ID.String(),
		UserID:       d.UserID.String(),
		Action:       d.Action,
		ResourceType: d.ResourceType,
		ResourceKey:  d.ResourceKey,
		Detail:       d.Detail,
		IP:           d.IP,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}
