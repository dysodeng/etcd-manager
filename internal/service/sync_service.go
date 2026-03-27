package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/etcd"
)

type SyncService struct {
	etcdClient   *etcd.Client
	envRepo      domain.EnvironmentRepository
	revisionRepo domain.ConfigRevisionRepository
}

func NewSyncService(
	etcdClient *etcd.Client,
	envRepo domain.EnvironmentRepository,
	revisionRepo domain.ConfigRevisionRepository,
) *SyncService {
	return &SyncService{etcdClient: etcdClient, envRepo: envRepo, revisionRepo: revisionRepo}
}

// EnvSyncStatus 环境同步状态
type EnvSyncStatus struct {
	EnvironmentID   string `json:"environment_id"`
	EnvironmentName string `json:"environment_name"`
	EtcdKeyCount    int    `json:"etcd_key_count"`
	DBKeyCount      int    `json:"db_key_count"`
	NeedRestore     bool   `json:"need_restore"`
}

// Check 检查各环境在 etcd 中是否缺失配置
func (s *SyncService) Check(ctx context.Context) ([]EnvSyncStatus, error) {
	envs, err := s.envRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]EnvSyncStatus, 0, len(envs))
	for _, env := range envs {
		status := EnvSyncStatus{
			EnvironmentID:   env.ID.String(),
			EnvironmentName: env.Name,
		}

		// 查 etcd 中该环境的配置 key 数量
		configPrefix := env.KeyPrefix + env.ConfigPrefix
		resp, err := s.etcdClient.GetWithPrefix(ctx, configPrefix, 0)
		if err == nil {
			status.EtcdKeyCount = len(resp.Kvs)
		}

		// 查 DB 中该环境的有效配置 key 数量（最新状态非 delete 的）
		latestRevs, err := s.revisionRepo.ListLatestByEnvironment(ctx, env.ID)
		if err == nil {
			for _, rev := range latestRevs {
				if rev.Action != "delete" {
					status.DBKeyCount++
				}
			}
		}

		// etcd 为空但 DB 有数据 → 需要恢复
		status.NeedRestore = status.EtcdKeyCount == 0 && status.DBKeyCount > 0
		result = append(result, status)
	}

	return result, nil
}

// RestoreResult 恢复结果
type RestoreResult struct {
	EnvironmentID   string   `json:"environment_id"`
	EnvironmentName string   `json:"environment_name"`
	Total           int      `json:"total"`
	Success         int      `json:"success"`
	Failed          []string `json:"failed,omitempty"`
}

// Restore 恢复选中环境的配置到 etcd
func (s *SyncService) Restore(ctx context.Context, envIDs []uuid.UUID) ([]RestoreResult, error) {
	results := make([]RestoreResult, 0, len(envIDs))

	for _, envID := range envIDs {
		env, err := s.envRepo.GetByID(ctx, envID)
		if err != nil {
			continue
		}

		result := RestoreResult{
			EnvironmentID:   env.ID.String(),
			EnvironmentName: env.Name,
		}

		// 获取该环境下每个 key 的最新 revision
		latestRevs, err := s.revisionRepo.ListLatestByEnvironment(ctx, env.ID)
		if err != nil {
			results = append(results, result)
			continue
		}

		for _, rev := range latestRevs {
			// 跳过已删除的 key
			if rev.Action == "delete" {
				continue
			}

			result.Total++
			fullKey := env.KeyPrefix + env.ConfigPrefix + rev.Key
			_, err := s.etcdClient.Put(ctx, fullKey, rev.Value)
			if err != nil {
				result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", rev.Key, err))
			} else {
				result.Success++
			}
		}

		results = append(results, result)
	}

	return results, nil
}
