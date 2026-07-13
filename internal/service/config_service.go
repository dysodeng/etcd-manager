package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/etcd"
)

type ConfigService struct {
	configStore  etcd.ConfigStore
	envRepo      domain.EnvironmentRepository
	revisionRepo domain.ConfigRevisionRepository
}

func NewConfigService(
	configStore etcd.ConfigStore,
	envRepo domain.EnvironmentRepository,
	revisionRepo domain.ConfigRevisionRepository,
) *ConfigService {
	return &ConfigService{
		configStore:  configStore,
		envRepo:      envRepo,
		revisionRepo: revisionRepo,
	}
}

type ConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *ConfigService) resolveAuthorizedEnvironment(ctx context.Context, envName string) (*domain.Environment, error) {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", envName)
	}
	if err := domain.RequireEnvironmentAccess(ctx, env.ID); err != nil {
		return nil, err
	}
	return env, nil
}

func (s *ConfigService) List(ctx context.Context, envName, prefix string) ([]ConfigItem, error) {
	env, err := s.resolveAuthorizedEnvironment(ctx, envName)
	if err != nil {
		return nil, err
	}
	configBase := env.KeyPrefix + env.ConfigPrefix
	fullPrefix := configBase + prefix
	resp, err := s.configStore.GetWithPrefix(ctx, fullPrefix, 0)
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
	if err := ValidateConfig(key, value); err != nil {
		return err
	}
	env, err := s.resolveAuthorizedEnvironment(ctx, envName)
	if err != nil {
		return err
	}
	fullKey := env.KeyPrefix + env.ConfigPrefix + key
	result, err := s.configStore.CreateIfAbsent(ctx, fullKey, value)
	if err != nil {
		return err
	}
	if !result.Succeeded {
		return ErrKeyExists
	}
	revision := &domain.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		Value:         value,
		EtcdRevision:  result.Revision,
		Action:        "create",
		Operator:      operatorID,
		Comment:       comment,
	}
	return s.persistRevisionWithCompensation(ctx, "create", fullKey, etcd.ConfigSnapshot{}, result.Revision, revision)
}

func (s *ConfigService) Update(ctx context.Context, envName, key, value, comment string, operatorID uuid.UUID) error {
	if err := ValidateConfig(key, value); err != nil {
		return err
	}
	env, err := s.resolveAuthorizedEnvironment(ctx, envName)
	if err != nil {
		return err
	}
	return s.upsertConfig(ctx, env, key, value, comment, operatorID, "update", "update")
}

func (s *ConfigService) Delete(ctx context.Context, envName, key string, operatorID uuid.UUID) error {
	env, err := s.resolveAuthorizedEnvironment(ctx, envName)
	if err != nil {
		return err
	}
	fullKey := env.KeyPrefix + env.ConfigPrefix + key
	before, err := s.configStore.GetConfig(ctx, fullKey)
	if err != nil {
		return err
	}
	if !before.Exists {
		return ErrKeyNotFound
	}
	result, err := s.configStore.DeleteIfModRevision(ctx, fullKey, before.ModRevision)
	if err != nil {
		return err
	}
	if !result.Succeeded {
		return ErrConfigConflict
	}
	revision := &domain.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		PrevValue:     before.Value,
		EtcdRevision:  result.Revision,
		Action:        "delete",
		Operator:      operatorID,
	}
	return s.persistRevisionWithCompensation(ctx, "delete", fullKey, before, result.Revision, revision)
}

func (s *ConfigService) Revisions(ctx context.Context, envName, key string, page, pageSize int) ([]domain.ConfigRevision, int64, error) {
	env, err := s.resolveAuthorizedEnvironment(ctx, envName)
	if err != nil {
		return nil, 0, err
	}
	return s.revisionRepo.ListByKey(ctx, env.ID, key, page, pageSize)
}

func (s *ConfigService) Rollback(ctx context.Context, envName, key string, revisionID uuid.UUID, operatorID uuid.UUID) error {
	env, err := s.resolveAuthorizedEnvironment(ctx, envName)
	if err != nil {
		return err
	}
	rev, err := s.revisionRepo.GetByID(ctx, revisionID)
	if err != nil {
		return ErrRevisionNotFound
	}
	if rev.EnvironmentID != env.ID || rev.Key != key || rev.Action == "delete" {
		return ErrRevisionNotFound
	}
	if err := ValidateConfig(key, rev.Value); err != nil {
		return err
	}
	return s.upsertConfig(ctx, env, key, rev.Value, fmt.Sprintf("rollback to revision %s", revisionID), operatorID, "update", "update")
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
	env, err := s.resolveAuthorizedEnvironment(ctx, envName)
	if err != nil {
		return nil, err
	}
	var configs map[string]string
	if err := json.Unmarshal(data, &configs); err != nil {
		if err := yaml.Unmarshal(data, &configs); err != nil {
			return nil, errors.New("invalid import format, expected JSON or YAML")
		}
	}
	result := &ImportResult{Total: len(configs)}
	validConfigs := make(map[string]string, len(configs))
	for key, value := range configs {
		if err := ValidateConfig(key, value); err != nil {
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		validConfigs[key] = value
	}
	if dryRun {
		result.Success = len(validConfigs)
		return result, nil
	}
	for key, value := range validConfigs {
		if err := s.upsertConfig(ctx, env, key, value, "import", operatorID, "update", "create"); err != nil {
			result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		result.Success++
	}
	return result, nil
}

func (s *ConfigService) upsertConfig(
	ctx context.Context,
	env *domain.Environment,
	key, value, comment string,
	operatorID uuid.UUID,
	existingAction, absentAction string,
) error {
	fullKey := env.KeyPrefix + env.ConfigPrefix + key
	before, err := s.configStore.GetConfig(ctx, fullKey)
	if err != nil {
		return err
	}
	action := absentAction
	var result etcd.ConditionalResult
	if before.Exists {
		action = existingAction
		result, err = s.configStore.PutIfModRevision(ctx, fullKey, value, before.ModRevision)
	} else {
		result, err = s.configStore.CreateIfAbsent(ctx, fullKey, value)
	}
	if err != nil {
		return err
	}
	if !result.Succeeded {
		return ErrConfigConflict
	}
	revision := &domain.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		Value:         value,
		PrevValue:     before.Value,
		EtcdRevision:  result.Revision,
		Action:        action,
		Operator:      operatorID,
		Comment:       comment,
	}
	return s.persistRevisionWithCompensation(ctx, action, fullKey, before, result.Revision, revision)
}

func (s *ConfigService) persistRevisionWithCompensation(
	ctx context.Context,
	operation, fullKey string,
	before etcd.ConfigSnapshot,
	writtenRevision int64,
	revision *domain.ConfigRevision,
) error {
	if err := s.revisionRepo.Create(ctx, revision); err != nil {
		var compensated bool
		var compensationErr error
		switch {
		case operation == "delete":
			compensated, compensationErr = s.configStore.RestoreIfAbsent(ctx, fullKey, before)
		case before.Exists:
			compensated, compensationErr = s.configStore.RestoreIfModRevision(ctx, fullKey, before, writtenRevision)
		default:
			compensated, compensationErr = s.configStore.DeleteIfModRevisionForCompensation(ctx, fullKey, writtenRevision)
		}
		return &ConfigPersistenceError{
			Operation:       operation,
			Err:             err,
			Compensated:     compensated,
			CompensationErr: compensationErr,
		}
	}
	return nil
}
