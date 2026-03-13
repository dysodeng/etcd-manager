package sqlite

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"gorm.io/gorm"
)

type EnvironmentRepository struct{ db *gorm.DB }

func NewEnvironmentRepository(db *gorm.DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

func (r *EnvironmentRepository) Create(ctx context.Context, env *model.Environment) error {
	return GetDB(ctx, r.db).Create(env).Error
}
func (r *EnvironmentRepository) GetByID(ctx context.Context, id uint) (*model.Environment, error) {
	var e model.Environment
	return &e, GetDB(ctx, r.db).First(&e, id).Error
}
func (r *EnvironmentRepository) GetByName(ctx context.Context, name string) (*model.Environment, error) {
	var e model.Environment
	return &e, GetDB(ctx, r.db).Where("name = ?", name).First(&e).Error
}
func (r *EnvironmentRepository) List(ctx context.Context) ([]model.Environment, error) {
	var envs []model.Environment
	return envs, GetDB(ctx, r.db).Order("sort_order ASC, id ASC").Find(&envs).Error
}
func (r *EnvironmentRepository) Update(ctx context.Context, env *model.Environment) error {
	return GetDB(ctx, r.db).Save(env).Error
}
func (r *EnvironmentRepository) Delete(ctx context.Context, id uint) error {
	return GetDB(ctx, r.db).Delete(&model.Environment{}, id).Error
}
