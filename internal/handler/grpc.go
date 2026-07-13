package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/service"
)

type GrpcHandler struct {
	grpcSvc  *service.GrpcServiceManager
	envSvc   *service.EnvironmentService
	auditSvc *service.AuditService
}

func NewGrpcHandler(grpcSvc *service.GrpcServiceManager, envSvc *service.EnvironmentService, auditSvc *service.AuditService) *GrpcHandler {
	return &GrpcHandler{grpcSvc: grpcSvc, envSvc: envSvc, auditSvc: auditSvc}
}

// List 列出所有 gRPC 服务（按服务名分组）
func (h *GrpcHandler) List(c *gin.Context) {
	envName := c.Query("env")
	if envName == "" {
		Fail(c, CodeParamInvalid, "env is required")
		return
	}
	env, err := h.envSvc.GetByName(c.Request.Context(), envName)
	if err != nil {
		Fail(c, CodeParamInvalid, "environment not found")
		return
	}
	groups, err := h.grpcSvc.ListServices(c.Request.Context(), env)
	if err != nil {
		code := CodeEtcdConnFailed
		if errors.Is(err, domain.ErrEnvironmentForbidden) {
			code = CodeForbidden
		}
		Fail(c, code, err.Error())
		return
	}
	OK(c, groups)
}

// UpdateStatus 更新实例状态（上线/下线）
func (h *GrpcHandler) UpdateStatus(c *gin.Context) {
	var req struct {
		Env    string `json:"env" binding:"required"`
		Key    string `json:"key" binding:"required"`
		Status string `json:"status" binding:"required,oneof=up down"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "env, key and status(up/down) are required")
		return
	}
	env, err := h.envSvc.GetByName(c.Request.Context(), req.Env)
	if err != nil {
		Fail(c, CodeParamInvalid, "environment not found")
		return
	}
	if err := h.grpcSvc.UpdateInstanceStatus(c.Request.Context(), env, req.Key, req.Status); err != nil {
		code := CodeEtcdConnFailed
		if errors.Is(err, domain.ErrEnvironmentForbidden) {
			code = CodeForbidden
		}
		Fail(c, code, err.Error())
		return
	}

	action := "deregister"
	if req.Status == "up" {
		action = "register"
	}
	userID, _ := getUserID(c)
	h.auditSvc.Log(c.Request.Context(), userID, action, "grpc_service_instance", req.Key, req.Status, c.ClientIP())

	OK(c, nil)
}
