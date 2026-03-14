package pgsql

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type EnvironmentRepository struct{ db *gorm.DB }

func NewEnvironmentRepository(db *gorm.DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

func (r *EnvironmentRepository) Create(ctx context.Context, env *domain.Environment) error {
	m := envToModel(env)
	if err := GetDB(ctx, r.db).Create(m).Error; err != nil {
		return err
	}
	env.ID = m.ID
	env.CreatedAt = m.CreatedAt
	env.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *EnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Environment, error) {
	var m Environment
	if err := GetDB(ctx, r.db).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return envToDomain(&m), nil
}

func (r *EnvironmentRepository) GetByName(ctx context.Context, name string) (*domain.Environment, error) {
	var m Environment
	if err := GetDB(ctx, r.db).Where("name = ?", name).First(&m).Error; err != nil {
		return nil, err
	}
	return envToDomain(&m), nil
}

func (r *EnvironmentRepository) List(ctx context.Context) ([]domain.Environment, error) {
	var models []Environment
	if err := GetDB(ctx, r.db).Order("sort_order ASC, id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	envs := make([]domain.Environment, len(models))
	for i := range models {
		envs[i] = *envToDomain(&models[i])
	}
	return envs, nil
}

func (r *EnvironmentRepository) Update(ctx context.Context, env *domain.Environment) error {
	return GetDB(ctx, r.db).Save(envToModel(env)).Error
}

func (r *EnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return GetDB(ctx, r.db).Where("id = ?", id).Delete(&Environment{}).Error
}

func envToDomain(m *Environment) *domain.Environment {
	return &domain.Environment{
		ID:            m.ID,
		Name:          m.Name,
		KeyPrefix:     m.KeyPrefix,
		ConfigPrefix:  m.ConfigPrefix,
		GatewayPrefix: m.GatewayPrefix,
		GrpcPrefix:    m.GrpcPrefix,
		Description:   m.Description,
		SortOrder:     m.SortOrder,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func envToModel(d *domain.Environment) *Environment {
	return &Environment{
		ID:            d.ID,
		Name:          d.Name,
		KeyPrefix:     d.KeyPrefix,
		ConfigPrefix:  d.ConfigPrefix,
		GatewayPrefix: d.GatewayPrefix,
		GrpcPrefix:    d.GrpcPrefix,
		Description:   d.Description,
		SortOrder:     d.SortOrder,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}
