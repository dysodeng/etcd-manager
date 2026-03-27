package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type SyncHandler struct {
	syncSvc  *service.SyncService
	auditSvc *service.AuditService
}

func NewSyncHandler(syncSvc *service.SyncService, auditSvc *service.AuditService) *SyncHandler {
	return &SyncHandler{syncSvc: syncSvc, auditSvc: auditSvc}
}

func (h *SyncHandler) Check(c *gin.Context) {
	statuses, err := h.syncSvc.Check(c.Request.Context())
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, statuses)
}

func (h *SyncHandler) Restore(c *gin.Context) {
	var req struct {
		EnvironmentIDs []string `json:"environment_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}

	envIDs := make([]uuid.UUID, 0, len(req.EnvironmentIDs))
	for _, s := range req.EnvironmentIDs {
		if id, err := uuid.Parse(s); err == nil {
			envIDs = append(envIDs, id)
		}
	}
	if len(envIDs) == 0 {
		Fail(c, CodeParamInvalid, "no valid environment IDs")
		return
	}

	results, err := h.syncSvc.Restore(c.Request.Context(), envIDs)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}

	operatorID, ok := getUserID(c)
	if ok {
		for _, r := range results {
			h.auditSvc.Log(c.Request.Context(), operatorID, "restore", "sync", r.EnvironmentName, "", c.ClientIP())
		}
	}

	OK(c, results)
}
