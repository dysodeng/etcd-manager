package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type KVHandler struct {
	kvSvc    *service.KVService
	auditSvc *service.AuditService
}

func NewKVHandler(kvSvc *service.KVService, auditSvc *service.AuditService) *KVHandler {
	return &KVHandler{kvSvc: kvSvc, auditSvc: auditSvc}
}

func (h *KVHandler) Get(c *gin.Context) {
	key := c.Query("key")
	prefix := c.Query("prefix")

	if key != "" {
		item, err := h.kvSvc.Get(c.Request.Context(), key)
		if err != nil {
			Fail(c, CodeEtcdOpFailed, err.Error())
			return
		}
		if item == nil {
			Fail(c, CodeKeyNotFound, "key not found")
			return
		}
		OK(c, item)
		return
	}

	if prefix == "" {
		prefix = "/"
	}
	limit := int64(50)
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.kvSvc.List(c.Request.Context(), prefix, limit)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	OK(c, items)
}

func (h *KVHandler) Create(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "key is required")
		return
	}
	existing, _ := h.kvSvc.Get(c.Request.Context(), req.Key)
	if existing != nil {
		Fail(c, CodeKeyExists, "key already exists")
		return
	}
	if err := h.kvSvc.Put(c.Request.Context(), req.Key, req.Value); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "create", "kv", req.Key, "", c.ClientIP())
	OK(c, nil)
}

func (h *KVHandler) Update(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "key is required")
		return
	}
	if err := h.kvSvc.Put(c.Request.Context(), req.Key, req.Value); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "update", "kv", req.Key, "", c.ClientIP())
	OK(c, nil)
}

func (h *KVHandler) Delete(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		Fail(c, CodeParamInvalid, "key is required")
		return
	}
	if err := h.kvSvc.Delete(c.Request.Context(), key); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID, "delete", "kv", key, "", c.ClientIP())
	OK(c, nil)
}
