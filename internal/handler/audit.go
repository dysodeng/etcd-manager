package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/config-center/internal/domain"
	"github.com/dysodeng/config-center/internal/service"
)

type AuditHandler struct {
	auditSvc *service.AuditService
}

func NewAuditHandler(auditSvc *service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	filter := domain.AuditLogFilter{
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
	}
	if uid := c.Query("user_id"); uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			filter.UserID = &id
		}
	}
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			filter.StartTime = &t
		}
	}
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			filter.EndTime = &t
		}
	}

	logs, total, err := h.auditSvc.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OKPage(c, logs, total, page, pageSize)
}
