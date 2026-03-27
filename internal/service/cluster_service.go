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
	DBSizeInUse int64           `json:"db_size_in_use"`
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
			metrics.DBSizeInUse = sr.DbSizeInUse
			if name, ok := memberNames[sr.Leader]; ok && name != "" {
				metrics.LeaderName = name
			} else {
				metrics.LeaderName = fmt.Sprintf("%x", sr.Leader)
			}
		}
	}
	return metrics, nil
}

// MemberStatus 单个成员的详细状态
type MemberStatus struct {
	Name             string `json:"name"`
	Endpoint         string `json:"endpoint"`
	DBSize           int64  `json:"db_size"`            // DB 文件总大小
	DBSizeInUse      int64  `json:"db_size_in_use"`     // DB 实际使用大小，差值为碎片空间
	Version          string `json:"version"`
	RaftIndex        uint64 `json:"raft_index"`          // Raft 日志最新条目索引，代表集群收到的写操作总序号
	RaftTerm         uint64 `json:"raft_term"`           // Raft 选举任期号，每次 Leader 选举 +1
	RaftAppliedIndex uint64 `json:"raft_applied_index"`  // 已应用到状态机的日志索引，正常时应接近 RaftIndex
	IsLearner        bool   `json:"is_learner"`          // Learner: 只读追随者，同步数据但不参与投票，用于安全扩容
	IsLeader         bool   `json:"is_leader"`
}

func (s *ClusterService) MemberStatuses(ctx context.Context) ([]MemberStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	memberResp, err := s.etcdClient.MemberList(ctx)
	if err != nil {
		return nil, err
	}

	// 建立 member ID → member 映射
	type memberMeta struct {
		Name      string
		IsLearner bool
	}
	idToMember := make(map[uint64]memberMeta)
	for _, m := range memberResp.Members {
		idToMember[m.ID] = memberMeta{Name: m.Name, IsLearner: m.IsLearner}
	}

	var results []MemberStatus
	endpoints := s.etcdClient.Endpoints()
	for _, ep := range endpoints {
		sr, err := s.etcdClient.Status(ctx, ep)
		if err != nil {
			continue
		}
		meta := idToMember[sr.Header.MemberId]
		results = append(results, MemberStatus{
			Name:             meta.Name,
			Endpoint:         ep,
			DBSize:           sr.DbSize,
			DBSizeInUse:      sr.DbSizeInUse,
			Version:          sr.Version,
			RaftIndex:        sr.RaftIndex,
			RaftTerm:         sr.RaftTerm,
			RaftAppliedIndex: sr.RaftAppliedIndex,
			IsLearner:        meta.IsLearner,
			IsLeader:         sr.Leader == sr.Header.MemberId,
		})
	}
	return results, nil
}

// AlarmInfo 报警信息
// NOSPACE: 磁盘空间不足，etcd 将拒绝写入
// CORRUPT: 数据损坏，需要从备份恢复
type AlarmInfo struct {
	MemberID  string `json:"member_id"`
	AlarmType string `json:"alarm_type"`
}

func (s *ClusterService) Alarms(ctx context.Context) ([]AlarmInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.etcdClient.AlarmList(ctx)
	if err != nil {
		return nil, err
	}

	alarms := make([]AlarmInfo, 0, len(resp.Alarms))
	for _, a := range resp.Alarms {
		alarmType := "UNKNOWN"
		switch a.Alarm {
		case 1:
			alarmType = "NOSPACE"
		case 2:
			alarmType = "CORRUPT"
		}
		alarms = append(alarms, AlarmInfo{
			MemberID:  fmt.Sprintf("%x", a.MemberID),
			AlarmType: alarmType,
		})
	}
	return alarms, nil
}
