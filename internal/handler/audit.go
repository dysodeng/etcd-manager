package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/service"
)

type AuditHandler struct {
	auditSvc *service.AuditService
	userSvc  *service.UserService
}

func NewAuditHandler(auditSvc *service.AuditService, userSvc *service.UserService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc, userSvc: userSvc}
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

	// 批量填充用户名
	userMap := make(map[uuid.UUID]string)
	for i := range logs {
		uid := logs[i].UserID
		if _, ok := userMap[uid]; !ok {
			if u, err := h.userSvc.GetByID(c.Request.Context(), uid); err == nil {
				userMap[uid] = u.Username
			}
		}
		logs[i].Username = userMap[uid]
	}

	OKPage(c, logs, total, page, pageSize)
}
