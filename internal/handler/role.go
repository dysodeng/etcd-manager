package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/service"
)

type RoleHandler struct {
	roleSvc  *service.RoleService
	auditSvc *service.AuditService
}

func NewRoleHandler(roleSvc *service.RoleService, auditSvc *service.AuditService) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc, auditSvc: auditSvc}
}

func (h *RoleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	roles, total, err := h.roleSvc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OKPage(c, roles, total, page, pageSize)
}

func (h *RoleHandler) GetByID(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "id")
	if !ok {
		return
	}
	role, err := h.roleSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, role)
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	svcReq := req.toServiceRequest()
	role, err := h.roleSvc.Create(c.Request.Context(), svcReq)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	operatorID, ok := getUserID(c)
	if ok {
		h.auditSvc.Log(c.Request.Context(), operatorID, "create", "role", req.Name, "", c.ClientIP())
	}
	OK(c, role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "id")
	if !ok {
		return
	}
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	svcReq := req.toServiceRequest()
	if err := h.roleSvc.Update(c.Request.Context(), id, svcReq); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	operatorID, ok := getUserID(c)
	if ok {
		h.auditSvc.Log(c.Request.Context(), operatorID, "update", "role", req.Name, "", c.ClientIP())
	}
	OK(c, nil)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "id")
	if !ok {
		return
	}
	if err := h.roleSvc.Delete(c.Request.Context(), id); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	operatorID, ok := getUserID(c)
	if ok {
		h.auditSvc.Log(c.Request.Context(), operatorID, "delete", "role", id.String(), "", c.ClientIP())
	}
	OK(c, nil)
}

type permissionRequest struct {
	Module   string `json:"module"`
	CanRead  bool   `json:"can_read"`
	CanWrite bool   `json:"can_write"`
}

type roleRequest struct {
	Name           string              `json:"name" binding:"required"`
	Description    string              `json:"description"`
	Permissions    []permissionRequest `json:"permissions"`
	EnvironmentIDs []string            `json:"environment_ids"`
}

func (r *roleRequest) toServiceRequest() *service.RoleCreateRequest {
	perms := make([]domain.RolePermission, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = domain.RolePermission{
			Module:   p.Module,
			CanRead:  p.CanRead,
			CanWrite: p.CanWrite,
		}
	}
	envIDs := make([]uuid.UUID, 0, len(r.EnvironmentIDs))
	for _, s := range r.EnvironmentIDs {
		if id, err := uuid.Parse(s); err == nil {
			envIDs = append(envIDs, id)
		}
	}
	return &service.RoleCreateRequest{
		Name:           r.Name,
		Description:    r.Description,
		Permissions:    perms,
		EnvironmentIDs: envIDs,
	}
}
