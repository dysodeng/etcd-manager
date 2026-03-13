package service

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
)

type AuditService struct {
	auditRepo store.AuditLogRepository
}

func NewAuditService(auditRepo store.AuditLogRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

func (s *AuditService) Log(ctx context.Context, userID uint, action, resourceType, resourceKey, detail, ip string) {
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceKey:  resourceKey,
		Detail:       detail,
		IP:           ip,
	})
}

func (s *AuditService) List(ctx context.Context, filter store.AuditLogFilter, page, pageSize int) ([]model.AuditLog, int64, error) {
	return s.auditRepo.List(ctx, filter, page, pageSize)
}
