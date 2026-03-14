package handler

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type ConfigCenterHandler struct {
	configSvc *service.ConfigService
	envSvc    *service.EnvironmentService
	auditSvc  *service.AuditService
}

func NewConfigCenterHandler(
	configSvc *service.ConfigService,
	envSvc *service.EnvironmentService,
	auditSvc *service.AuditService,
) *ConfigCenterHandler {
	return &ConfigCenterHandler{configSvc: configSvc, envSvc: envSvc, auditSvc: auditSvc}
}

func (h *ConfigCenterHandler) ListEnvironments(c *gin.Context) {
	envs, err := h.envSvc.List(c.Request.Context())
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, envs)
}

func (h *ConfigCenterHandler) CreateEnvironment(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		KeyPrefix     string `json:"key_prefix" binding:"required"`
		ConfigPrefix  string `json:"config_prefix"`
		GatewayPrefix string `json:"gateway_prefix"`
		GrpcPrefix    string `json:"grpc_prefix"`
		Description   string `json:"description"`
		SortOrder     int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	if req.ConfigPrefix == "" {
		req.ConfigPrefix = "config/"
	}
	if req.GatewayPrefix == "" {
		req.GatewayPrefix = "gw-services/"
	}
	if req.GrpcPrefix == "" {
		req.GrpcPrefix = "grpc-services/"
	}
	env, err := h.envSvc.Create(c.Request.Context(), req.Name, req.KeyPrefix, req.ConfigPrefix, req.GatewayPrefix, req.GrpcPrefix, req.Description, req.SortOrder)
	if err != nil {
		Fail(c, CodeEnvExists, err.Error())
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "create", "environment", req.Name, "", c.ClientIP())
	OK(c, env)
}

func (h *ConfigCenterHandler) UpdateEnvironment(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "id")
	if !ok {
		return
	}
	var req struct {
		Name          string `json:"name" binding:"required"`
		KeyPrefix     string `json:"key_prefix" binding:"required"`
		ConfigPrefix  string `json:"config_prefix"`
		GatewayPrefix string `json:"gateway_prefix"`
		GrpcPrefix    string `json:"grpc_prefix"`
		Description   string `json:"description"`
		SortOrder     int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	if err := h.envSvc.Update(c.Request.Context(), id, req.Name, req.KeyPrefix, req.ConfigPrefix, req.GatewayPrefix, req.GrpcPrefix, req.Description, req.SortOrder); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, nil)
}

func (h *ConfigCenterHandler) DeleteEnvironment(c *gin.Context) {
	id, ok := parseUUID(c, c.Param("id"), "id")
	if !ok {
		return
	}
	if err := h.envSvc.Delete(c.Request.Context(), id); err != nil {
		if err.Error() == "environment has configs, cannot delete" {
			Fail(c, CodeEnvHasConfigs, err.Error())
		} else {
			Fail(c, CodeEtcdOpFailed, err.Error())
		}
		return
	}
	OK(c, nil)
}

func (h *ConfigCenterHandler) ListConfigs(c *gin.Context) {
	env := c.Query("env")
	prefix := c.Query("prefix")
	if env == "" {
		Fail(c, CodeParamInvalid, "env is required")
		return
	}
	items, err := h.configSvc.List(c.Request.Context(), env, prefix)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	OK(c, items)
}

func (h *ConfigCenterHandler) CreateConfig(c *gin.Context) {
	var req struct {
		Env     string `json:"env" binding:"required"`
		Key     string `json:"key" binding:"required"`
		Value   string `json:"value"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.configSvc.Create(c.Request.Context(), req.Env, req.Key, req.Value, req.Comment, userID); err != nil {
		if err.Error() == "key already exists" {
			Fail(c, CodeKeyExists, err.Error())
		} else {
			Fail(c, CodeEtcdOpFailed, err.Error())
		}
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "create", "config", req.Key, req.Env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Env     string `json:"env" binding:"required"`
		Key     string `json:"key" binding:"required"`
		Value   string `json:"value"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.configSvc.Update(c.Request.Context(), req.Env, req.Key, req.Value, req.Comment, userID); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "update", "config", req.Key, req.Env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) DeleteConfig(c *gin.Context) {
	env := c.Query("env")
	key := c.Query("key")
	if env == "" || key == "" {
		Fail(c, CodeParamInvalid, "env and key are required")
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.configSvc.Delete(c.Request.Context(), env, key, userID); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "delete", "config", key, env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) Revisions(c *gin.Context) {
	env := c.Query("env")
	key := c.Query("key")
	if env == "" || key == "" {
		Fail(c, CodeParamInvalid, "env and key are required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	revs, total, err := h.configSvc.Revisions(c.Request.Context(), env, key, page, pageSize)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	OKPage(c, revs, total, page, pageSize)
}

func (h *ConfigCenterHandler) Rollback(c *gin.Context) {
	var req struct {
		Env        string `json:"env" binding:"required"`
		Key        string `json:"key" binding:"required"`
		RevisionID string `json:"revision_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	revID, ok := parseUUID(c, req.RevisionID, "revision_id")
	if !ok {
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.configSvc.Rollback(c.Request.Context(), req.Env, req.Key, revID, userID); err != nil {
		if err.Error() == "revision not found" {
			Fail(c, CodeRevisionNotFound, err.Error())
		} else {
			Fail(c, CodeEtcdOpFailed, err.Error())
		}
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "rollback", "config", req.Key, req.Env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) Export(c *gin.Context) {
	env := c.Query("env")
	format := c.DefaultQuery("format", "json")
	if env == "" {
		Fail(c, CodeParamInvalid, "env is required")
		return
	}
	data, err := h.configSvc.Export(c.Request.Context(), env, format)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=config-"+env+"."+format)
	c.Data(200, "application/octet-stream", data)
}

func (h *ConfigCenterHandler) Import(c *gin.Context) {
	env := c.Query("env")
	dryRun := c.Query("dry_run") == "true"
	if env == "" {
		Fail(c, CodeParamInvalid, "env is required")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		Fail(c, CodeImportFormat, "failed to read request body")
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	result, err := h.configSvc.Import(c.Request.Context(), env, body, dryRun, userID)
	if err != nil {
		Fail(c, CodeImportFormat, err.Error())
		return
	}
	if len(result.Failed) > 0 {
		c.JSON(200, Response{Code: CodeImportPartial, Message: "partial import", Data: result})
		return
	}
	OK(c, result)
}
