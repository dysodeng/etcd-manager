package service

import (
	"context"
	"errors"

	"github.com/dysodeng/config-center/internal/etcd"
	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
)

type EnvironmentService struct {
	envRepo    store.EnvironmentRepository
	etcdClient *etcd.Client
}

func NewEnvironmentService(envRepo store.EnvironmentRepository, etcdClient *etcd.Client) *EnvironmentService {
	return &EnvironmentService{envRepo: envRepo, etcdClient: etcdClient}
}

func (s *EnvironmentService) Create(ctx context.Context, name, keyPrefix, description string, sortOrder int) (*model.Environment, error) {
	if _, err := s.envRepo.GetByName(ctx, name); err == nil {
		return nil, errors.New("environment already exists")
	}
	env := &model.Environment{
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

func (s *EnvironmentService) List(ctx context.Context) ([]model.Environment, error) {
	return s.envRepo.List(ctx)
}

func (s *EnvironmentService) Update(ctx context.Context, id uint, name, keyPrefix, description string, sortOrder int) error {
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

func (s *EnvironmentService) Delete(ctx context.Context, id uint) error {
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

func (s *EnvironmentService) GetByID(ctx context.Context, id uint) (*model.Environment, error) {
	return s.envRepo.GetByID(ctx, id)
}

func (s *EnvironmentService) GetByName(ctx context.Context, name string) (*model.Environment, error) {
	return s.envRepo.GetByName(ctx, name)
}
