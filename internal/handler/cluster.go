package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type ClusterHandler struct {
	clusterSvc *service.ClusterService
}

func NewClusterHandler(clusterSvc *service.ClusterService) *ClusterHandler {
	return &ClusterHandler{clusterSvc: clusterSvc}
}

func (h *ClusterHandler) Status(c *gin.Context) {
	status, err := h.clusterSvc.Status(c.Request.Context())
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, status)
}

func (h *ClusterHandler) Metrics(c *gin.Context) {
	metrics, err := h.clusterSvc.Metrics(c.Request.Context())
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, metrics)
}

func (h *ClusterHandler) MemberStatuses(c *gin.Context) {
	statuses, err := h.clusterSvc.MemberStatuses(c.Request.Context())
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, statuses)
}

func (h *ClusterHandler) Alarms(c *gin.Context) {
	alarms, err := h.clusterSvc.Alarms(c.Request.Context())
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, alarms)
}
