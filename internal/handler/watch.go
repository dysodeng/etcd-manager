package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dysodeng/etcd-manager/internal/etcd"
)

type WatchHandler struct {
	etcdClient *etcd.Client
}

func NewWatchHandler(etcdClient *etcd.Client) *WatchHandler {
	return &WatchHandler{etcdClient: etcdClient}
}

type WatchEvent struct {
	Type     string `json:"type"`
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	Revision int64  `json:"revision"`
}

func (h *WatchHandler) Watch(c *gin.Context) {
	prefix := c.Query("prefix")
	if prefix == "" {
		Fail(c, CodeParamInvalid, "prefix is required")
		return
	}

	var startRev int64
	if lastID := c.GetHeader("Last-Event-ID"); lastID != "" {
		if rev, err := strconv.ParseInt(lastID, 10, 64); err == nil {
			startRev = rev + 1
		}
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx := c.Request.Context()
	watchCh := h.etcdClient.Watch(ctx, prefix, startRev)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(30 * time.Minute)
	defer timeout.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		case <-timeout.C:
			return false
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			return true
		case resp, ok := <-watchCh:
			if !ok {
				return false
			}
			if resp.CompactRevision > 0 {
				evt := WatchEvent{Type: "COMPACTED", Revision: resp.CompactRevision}
				data, _ := json.Marshal(evt)
				fmt.Fprintf(w, "event: kv_change\ndata: %s\nid: %d\n\n", data, resp.CompactRevision)
				return true
			}
			for _, ev := range resp.Events {
				evt := WatchEvent{
					Key:      string(ev.Kv.Key),
					Revision: ev.Kv.ModRevision,
				}
				if ev.Type == clientv3.EventTypePut {
					evt.Type = "PUT"
					evt.Value = string(ev.Kv.Value)
				} else {
					evt.Type = "DELETE"
				}
				data, _ := json.Marshal(evt)
				fmt.Fprintf(w, "event: kv_change\ndata: %s\nid: %d\n\n", data, ev.Kv.ModRevision)
			}
			return true
		}
	})
}
