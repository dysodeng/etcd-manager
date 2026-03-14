package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/dysodeng/config-center/internal/domain"
	"github.com/dysodeng/config-center/internal/etcd"
)

type ConfigService struct {
	etcdClient   *etcd.Client
	envRepo      domain.EnvironmentRepository
	revisionRepo domain.ConfigRevisionRepository
	txManager    domain.TransactionManager
}

func NewConfigService(
	etcdClient *etcd.Client,
	envRepo domain.EnvironmentRepository,
	revisionRepo domain.ConfigRevisionRepository,
	txManager domain.TransactionManager,
) *ConfigService {
	return &ConfigService{
		etcdClient:   etcdClient,
		envRepo:      envRepo,
		revisionRepo: revisionRepo,
		txManager:    txManager,
	}
}

type ConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *ConfigService) List(ctx context.Context, envName, prefix string) ([]ConfigItem, error) {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", envName)
	}
	configBase := env.KeyPrefix + env.ConfigPrefix
	fullPrefix := configBase + prefix
	resp, err := s.etcdClient.GetWithPrefix(ctx, fullPrefix, 0)
	if err != nil {
		return nil, err
	}
	items := make([]ConfigItem, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		shortKey := strings.TrimPrefix(string(kv.Key), configBase)
		items = append(items, ConfigItem{Key: shortKey, Value: string(kv.Value)})
	}
	return items, nil
}

func (s *ConfigService) Create(ctx context.Context, envName, key, value, comment string, operatorID uuid.UUID) error {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}
	fullKey := env.KeyPrefix + env.ConfigPrefix + key
	existing, err := s.etcdClient.Get(ctx, fullKey)
	if err != nil {
		return err
	}
	if len(existing.Kvs) > 0 {
		return errors.New("key already exists")
	}
	resp, err := s.etcdClient.Put(ctx, fullKey, value)
	if err != nil {
		return err
	}
	return s.revisionRepo.Create(ctx, &domain.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		Value:         value,
		EtcdRevision:  resp.Header.Revision,
		Action:        "create",
		Operator:      operatorID,
		Comment:       comment,
	})
}

func (s *ConfigService) Update(ctx context.Context, envName, key, value, comment string, operatorID uuid.UUID) error {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}
	fullKey := env.KeyPrefix + env.ConfigPrefix + key
	existing, err := s.etcdClient.Get(ctx, fullKey)
	if err != nil {
		return err
	}
	var prevValue string
	if len(existing.Kvs) > 0 {
		prevValue = string(existing.Kvs[0].Value)
	}
	resp, err := s.etcdClient.Put(ctx, fullKey, value)
	if err != nil {
		return err
	}
	return s.revisionRepo.Create(ctx, &domain.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		Value:         value,
		PrevValue:     prevValue,
		EtcdRevision:  resp.Header.Revision,
		Action:        "update",
		Operator:      operatorID,
		Comment:       comment,
	})
}

func (s *ConfigService) Delete(ctx context.Context, envName, key string, operatorID uuid.UUID) error {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}
	fullKey := env.KeyPrefix + env.ConfigPrefix + key
	existing, err := s.etcdClient.Get(ctx, fullKey)
	if err != nil {
		return err
	}
	var prevValue string
	if len(existing.Kvs) > 0 {
		prevValue = string(existing.Kvs[0].Value)
	}
	resp, err := s.etcdClient.Delete(ctx, fullKey)
	if err != nil {
		return err
	}
	return s.revisionRepo.Create(ctx, &domain.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		PrevValue:     prevValue,
		EtcdRevision:  resp.Header.Revision,
		Action:        "delete",
		Operator:      operatorID,
	})
}

func (s *ConfigService) Revisions(ctx context.Context, envName, key string, page, pageSize int) ([]domain.ConfigRevision, int64, error) {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return nil, 0, fmt.Errorf("environment not found: %s", envName)
	}
	return s.revisionRepo.ListByKey(ctx, env.ID, key, page, pageSize)
}

func (s *ConfigService) Rollback(ctx context.Context, envName, key string, revisionID uuid.UUID, operatorID uuid.UUID) error {
	rev, err := s.revisionRepo.GetByID(ctx, revisionID)
	if err != nil {
		return errors.New("revision not found")
	}
	return s.Update(ctx, envName, key, rev.Value, fmt.Sprintf("rollback to revision %s", revisionID), operatorID)
}

func (s *ConfigService) Export(ctx context.Context, envName, format string) ([]byte, error) {
	items, err := s.List(ctx, envName, "")
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(items))
	for _, item := range items {
		m[item.Key] = item.Value
	}
	switch format {
	case "json":
		return json.MarshalIndent(m, "", "  ")
	case "yaml":
		return yaml.Marshal(m)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

type ImportResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  []string `json:"failed,omitempty"`
}

func (s *ConfigService) Import(ctx context.Context, envName string, data []byte, dryRun bool, operatorID uuid.UUID) (*ImportResult, error) {
	var configs map[string]string
	if err := json.Unmarshal(data, &configs); err != nil {
		if err := yaml.Unmarshal(data, &configs); err != nil {
			return nil, errors.New("invalid import format, expected JSON or YAML")
		}
	}
	result := &ImportResult{Total: len(configs)}
	if dryRun {
		result.Success = result.Total
		return result, nil
	}
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", envName)
	}
	for key, value := range configs {
		fullKey := env.KeyPrefix + env.ConfigPrefix + key
		existing, _ := s.etcdClient.Get(ctx, fullKey)
		action := "create"
		var prevValue string
		if len(existing.Kvs) > 0 {
			action = "update"
			prevValue = string(existing.Kvs[0].Value)
		}
		resp, err := s.etcdClient.Put(ctx, fullKey, value)
		if err != nil {
			result.Failed = append(result.Failed, key)
			continue
		}
		_ = s.revisionRepo.Create(ctx, &domain.ConfigRevision{
			EnvironmentID: env.ID,
			Key:           key,
			Value:         value,
			PrevValue:     prevValue,
			EtcdRevision:  resp.Header.Revision,
			Action:        action,
			Operator:      operatorID,
			Comment:       "import",
		})
		result.Success++
	}
	return result, nil
}
