package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/dysodeng/config-center/internal/domain"
	"github.com/dysodeng/config-center/internal/etcd"
)

type EnvironmentService struct {
	envRepo    domain.EnvironmentRepository
	etcdClient *etcd.Client
}

func NewEnvironmentService(envRepo domain.EnvironmentRepository, etcdClient *etcd.Client) *EnvironmentService {
	return &EnvironmentService{envRepo: envRepo, etcdClient: etcdClient}
}

func (s *EnvironmentService) Create(ctx context.Context, name, keyPrefix, description string, sortOrder int) (*domain.Environment, error) {
	if _, err := s.envRepo.GetByName(ctx, name); err == nil {
		return nil, errors.New("environment already exists")
	}
	env := &domain.Environment{
		Name:        name,
		KeyPrefix:   keyPrefix,
		Description: description,
		SortOrder:   sortOrder,
	}
	if err := s.envRepo.Create(ctx, env); err != nil {
		return nil, err
	}
	return env, nil
}

func (s *EnvironmentService) List(ctx context.Context) ([]domain.Environment, error) {
	return s.envRepo.List(ctx)
}

func (s *EnvironmentService) Update(ctx context.Context, id uuid.UUID, name, keyPrefix, description string, sortOrder int) error {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	env.Name = name
	env.KeyPrefix = keyPrefix
	env.Description = description
	env.SortOrder = sortOrder
	return s.envRepo.Update(ctx, env)
}

func (s *EnvironmentService) Delete(ctx context.Context, id uuid.UUID) error {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	resp, err := s.etcdClient.GetWithPrefix(ctx, env.KeyPrefix, 1)
	if err != nil {
		return err
	}
	if len(resp.Kvs) > 0 {
		return errors.New("environment has configs, cannot delete")
	}
	return s.envRepo.Delete(ctx, id)
}

func (s *EnvironmentService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Environment, error) {
	return s.envRepo.GetByID(ctx, id)
}

func (s *EnvironmentService) GetByName(ctx context.Context, name string) (*domain.Environment, error) {
	return s.envRepo.GetByName(ctx, name)
}
