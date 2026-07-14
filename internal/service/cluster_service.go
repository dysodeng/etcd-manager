package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dysodeng/etcd-manager/internal/etcd"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"
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

type memberProbeTarget struct {
	memberID uint64
	endpoint string
}

type memberProbeResult struct {
	memberID uint64
	endpoint string
	status   *clientv3.StatusResponse
	err      error
}

func uniqueEndpoints(endpoints []string) []string {
	unique := make([]string, 0, len(endpoints))
	seen := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint == "" {
			continue
		}
		if _, exists := seen[endpoint]; exists {
			continue
		}
		seen[endpoint] = struct{}{}
		unique = append(unique, endpoint)
	}
	return unique
}

func memberClientTargets(members []*etcdserverpb.Member, resolved map[uint64]memberProbeResult, usedEndpoints map[string]struct{}) []memberProbeTarget {
	memberURLs := make([][]string, len(members))
	owners := make(map[string]int, len(members))
	for i, member := range members {
		seen := make(map[string]struct{}, len(member.ClientURLs))
		for _, endpoint := range member.ClientURLs {
			if endpoint == "" {
				continue
			}
			if _, exists := seen[endpoint]; exists {
				continue
			}
			seen[endpoint] = struct{}{}
			memberURLs[i] = append(memberURLs[i], endpoint)
			owners[endpoint]++
		}
	}

	targets := make([]memberProbeTarget, 0, len(members)-len(resolved))
	for i, urls := range memberURLs {
		memberID := members[i].ID
		if _, exists := resolved[memberID]; exists {
			continue
		}
		chosen := ""
		for _, endpoint := range urls {
			if _, used := usedEndpoints[endpoint]; !used && owners[endpoint] == 1 {
				chosen = endpoint
				break
			}
		}
		if chosen == "" {
			for _, endpoint := range urls {
				if _, used := usedEndpoints[endpoint]; !used {
					chosen = endpoint
					break
				}
			}
		}
		if chosen != "" {
			usedEndpoints[chosen] = struct{}{}
			targets = append(targets, memberProbeTarget{memberID: memberID, endpoint: chosen})
		}
	}
	return targets
}

type endpointStatusResult struct {
	endpoint string
	status   *clientv3.StatusResponse
	err      error
}

func (s *ClusterService) probeEndpoints(ctx context.Context, endpoints []string) []endpointStatusResult {
	results := make([]endpointStatusResult, len(endpoints))
	var wg sync.WaitGroup
	wg.Add(len(endpoints))
	for i, endpoint := range endpoints {
		go func() {
			defer wg.Done()
			status, err := s.etcdClient.Status(ctx, endpoint)
			results[i] = endpointStatusResult{endpoint: endpoint, status: status, err: err}
		}()
	}
	wg.Wait()
	return results
}

func (s *ClusterService) probeMembers(ctx context.Context, members []*etcdserverpb.Member) []memberProbeResult {
	knownMembers := make(map[uint64]struct{}, len(members))
	for _, member := range members {
		knownMembers[member.ID] = struct{}{}
	}

	configuredEndpoints := uniqueEndpoints(s.etcdClient.Endpoints())
	usedEndpoints := make(map[string]struct{}, len(configuredEndpoints))
	for _, endpoint := range configuredEndpoints {
		usedEndpoints[endpoint] = struct{}{}
	}

	resolved := make(map[uint64]memberProbeResult, len(members))
	for _, result := range s.probeEndpoints(ctx, configuredEndpoints) {
		if result.err != nil || result.status == nil || result.status.Header == nil {
			continue
		}
		memberID := result.status.Header.MemberId
		if _, known := knownMembers[memberID]; !known {
			continue
		}
		if _, exists := resolved[memberID]; exists {
			continue
		}
		resolved[memberID] = memberProbeResult{
			memberID: memberID,
			endpoint: result.endpoint,
			status:   result.status,
		}
	}

	failed := make(map[uint64]memberProbeResult)
	targets := memberClientTargets(members, resolved, usedEndpoints)
	advertisedEndpoints := make([]string, len(targets))
	for i, target := range targets {
		advertisedEndpoints[i] = target.endpoint
	}
	for i, result := range s.probeEndpoints(ctx, advertisedEndpoints) {
		target := targets[i]
		probe := memberProbeResult{
			memberID: target.memberID,
			endpoint: result.endpoint,
			status:   result.status,
			err:      result.err,
		}
		if result.err != nil || result.status == nil || result.status.Header == nil {
			failed[target.memberID] = probe
			continue
		}
		if result.status.Header.MemberId != target.memberID {
			probe.err = fmt.Errorf("endpoint %s answered as member %x, want %x", result.endpoint, result.status.Header.MemberId, target.memberID)
			failed[target.memberID] = probe
			continue
		}
		resolved[target.memberID] = probe
	}

	results := make([]memberProbeResult, 0, len(members))
	for _, member := range members {
		if result, exists := resolved[member.ID]; exists {
			results = append(results, result)
			continue
		}
		if result, exists := failed[member.ID]; exists {
			results = append(results, result)
		}
	}
	return results
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

	metrics := &ClusterMetrics{
		ClusterID:   fmt.Sprintf("%x", memberResp.Header.ClusterId),
		MemberCount: len(memberResp.Members),
		Health:      make(map[string]bool),
	}
	for _, result := range s.probeMembers(ctx, memberResp.Members) {
		if result.err != nil {
			metrics.Health[result.endpoint] = false
			continue
		}
		sr := result.status
		metrics.Health[result.endpoint] = true
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
	DBSize           int64  `json:"db_size"`        // DB 文件总大小
	DBSizeInUse      int64  `json:"db_size_in_use"` // DB 实际使用大小，差值为碎片空间
	Version          string `json:"version"`
	RaftIndex        uint64 `json:"raft_index"`         // Raft 日志最新条目索引，代表集群收到的写操作总序号
	RaftTerm         uint64 `json:"raft_term"`          // Raft 选举任期号，每次 Leader 选举 +1
	RaftAppliedIndex uint64 `json:"raft_applied_index"` // 已应用到状态机的日志索引，正常时应接近 RaftIndex
	IsLearner        bool   `json:"is_learner"`         // Learner: 只读追随者，同步数据但不参与投票，用于安全扩容
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
	for _, result := range s.probeMembers(ctx, memberResp.Members) {
		if result.err != nil {
			continue
		}
		sr := result.status
		meta, exists := idToMember[sr.Header.MemberId]
		if !exists {
			continue
		}
		results = append(results, MemberStatus{
			Name:             meta.Name,
			Endpoint:         result.endpoint,
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
