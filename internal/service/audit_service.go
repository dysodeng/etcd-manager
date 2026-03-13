package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/dysodeng/config-center/internal/domain"
)

type AuditService struct {
	auditRepo domain.AuditLogRepository
}

func NewAuditService(auditRepo domain.AuditLogRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

func (s *AuditService) Log(ctx context.Context, userID uuid.UUID, action, resourceType, resourceKey, detail, ip string) {
	_ = s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceKey:  resourceKey,
		Detail:       detail,
		IP:           ip,
	})
}

func (s *AuditService) List(ctx context.Context, filter domain.AuditLogFilter, page, pageSize int) ([]domain.AuditLog, int64, error) {
	return s.auditRepo.List(ctx, filter, page, pageSize)
}
