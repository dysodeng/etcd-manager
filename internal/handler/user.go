package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type UserHandler struct {
	userSvc  *service.UserService
	auditSvc *service.AuditService
}

func NewUserHandler(userSvc *service.UserService, auditSvc *service.AuditService) *UserHandler {
	return &UserHandler{userSvc: userSvc, auditSvc: auditSvc}
}

func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	users, total, err := h.userSvc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OKPage(c, users, total, page, pageSize)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	user, err := h.userSvc.Create(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		if err.Error() == "username already exists" {
			Fail(c, CodeUserExists, err.Error())
		} else {
			Fail(c, CodeInternalError, err.Error())
		}
		return
	}
	operatorID, ok := getUserID(c)
	if !ok {
		return
	}
	h.auditSvc.Log(c.Request.Context(), operatorID, "create", "user", req.Username, "", c.ClientIP())
	OK(c, gin.H{"id": user.ID, "username": user.Username, "role": user.Role})
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "id")
	if !ok {
		return
	}
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	if err := h.userSvc.Update(c.Request.Context(), id, req.Role); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "id")
	if !ok {
		return
	}
	if err := h.userSvc.Delete(c.Request.Context(), id); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, nil)
}
