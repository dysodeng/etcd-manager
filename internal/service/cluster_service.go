package service

import (
	"context"
	"fmt"
	"time"

	"github.com/dysodeng/etcd-manager/internal/etcd"
)

type ClusterService struct {
	etcdClient *etcd.Client
}

func NewClusterService(etcdClient *etcd.Client) *ClusterService {
	return &ClusterService{etcdClient: etcdClient}
}

type MemberInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peer_urls"`
	ClientURLs []string `json:"client_urls"`
	IsLearner  bool     `json:"is_learner"`
}

type ClusterStatus struct {
	ClusterID string       `json:"cluster_id"`
	Members   []MemberInfo `json:"members"`
	Leader    string       `json:"leader"`
}

func (s *ClusterService) Status(ctx context.Context) (*ClusterStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.etcdClient.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	status := &ClusterStatus{
		ClusterID: fmt.Sprintf("%x", resp.Header.ClusterId),
	}
	memberNames := make(map[uint64]string)
	for _, m := range resp.Members {
		memberNames[m.ID] = m.Name
		status.Members = append(status.Members, MemberInfo{
			ID:         fmt.Sprintf("%x", m.ID),
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
			ClientURLs: m.ClientURLs,
			IsLearner:  m.IsLearner,
		})
	}
	endpoints := s.etcdClient.Endpoints()
	if len(endpoints) > 0 {
		sr, err := s.etcdClient.Status(ctx, endpoints[0])
		if err == nil {
			if name, ok := memberNames[sr.Leader]; ok && name != "" {
				status.Leader = name
			} else {
				status.Leader = fmt.Sprintf("%x", sr.Leader)
			}
		}
	}
	return status, nil
}

type ClusterMetrics struct {
	ClusterID   string          `json:"cluster_id"`
	Version     string          `json:"version"`
	DBSize      int64           `json:"db_size"`
	LeaderName  string          `json:"leader_name"`
	MemberCount int             `json:"member_count"`
	Health      map[string]bool `json:"health"`
}

func (s *ClusterService) Metrics(ctx context.Context) (*ClusterMetrics, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 先获取成员列表用于 ID→Name 映射
	memberResp, err := s.etcdClient.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	memberNames := make(map[uint64]string)
	for _, m := range memberResp.Members {
		memberNames[m.ID] = m.Name
	}

	endpoints := s.etcdClient.Endpoints()
	metrics := &ClusterMetrics{
		ClusterID:   fmt.Sprintf("%x", memberResp.Header.ClusterId),
		MemberCount: len(memberResp.Members),
		Health:      make(map[string]bool),
	}
	for _, ep := range endpoints {
		sr, err := s.etcdClient.Status(ctx, ep)
		if err != nil {
			metrics.Health[ep] = false
			continue
		}
		metrics.Health[ep] = true
		if metrics.Version == "" {
			metrics.Version = sr.Version
			metrics.DBSize = sr.DbSize
			if name, ok := memberNames[sr.Leader]; ok && name != "" {
				metrics.LeaderName = name
			} else {
				metrics.LeaderName = fmt.Sprintf("%x", sr.Leader)
			}
		}
	}
	return metrics, nil
}
