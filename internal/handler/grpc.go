package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type GrpcHandler struct {
	grpcSvc  *service.GrpcServiceManager
	auditSvc *service.AuditService
}

func NewGrpcHandler(grpcSvc *service.GrpcServiceManager, auditSvc *service.AuditService) *GrpcHandler {
	return &GrpcHandler{grpcSvc: grpcSvc, auditSvc: auditSvc}
}

// List 列出所有 gRPC 服务（按服务名分组）
func (h *GrpcHandler) List(c *gin.Context) {
	prefix := c.Query("prefix")
	if prefix == "" {
		Fail(c, CodeParamInvalid, "prefix is required")
		return
	}

	groups, err := h.grpcSvc.ListServices(c.Request.Context(), prefix)
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, groups)
}

// UpdateStatus 更新实例状态（上线/下线）
func (h *GrpcHandler) UpdateStatus(c *gin.Context) {
	var req struct {
		Key    string `json:"key" binding:"required"`
		Status string `json:"status" binding:"required,oneof=up down"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "key and status(up/down) are required")
		return
	}

	if err := h.grpcSvc.UpdateInstanceStatus(c.Request.Context(), req.Key, req.Status); err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
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
