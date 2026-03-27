package pgsql

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type RoleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) *RoleRepository { return &RoleRepository{db: db} }

func (r *RoleRepository) Create(ctx context.Context, role *domain.Role) error {
	return GetDB(ctx, r.db).Create(roleToModel(role)).Error
}

func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	var m Role
	if err := GetDB(ctx, r.db).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return roleToDomain(&m), nil
}

func (r *RoleRepository) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	var m Role
	if err := GetDB(ctx, r.db).Where("name = ?", name).First(&m).Error; err != nil {
		return nil, err
	}
	return roleToDomain(&m), nil
}

func (r *RoleRepository) List(ctx context.Context, page, pageSize int) ([]domain.Role, int64, error) {
	var models []Role
	var total int64
	db := GetDB(ctx, r.db).Model(&Role{})
	db.Count(&total)
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id ASC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	roles := make([]domain.Role, len(models))
	for i := range models {
		roles[i] = *roleToDomain(&models[i])
	}
	return roles, total, nil
}

func (r *RoleRepository) Update(ctx context.Context, role *domain.Role) error {
	return GetDB(ctx, r.db).Save(roleToModel(role)).Error
}

func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	db := GetDB(ctx, r.db)
	db.Where("role_id = ?", id).Delete(&RolePermission{})
	db.Where("role_id = ?", id).Delete(&RoleEnvironment{})
	return db.Where("id = ?", id).Delete(&Role{}).Error
}

func (r *RoleRepository) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]domain.RolePermission, error) {
	var models []RolePermission
	if err := GetDB(ctx, r.db).Where("role_id = ?", roleID).Find(&models).Error; err != nil {
		return nil, err
	}
	perms := make([]domain.RolePermission, len(models))
	for i, m := range models {
		perms[i] = domain.RolePermission{
			ID:       m.ID,
			RoleID:   m.RoleID,
			Module:   m.Module,
			CanRead:  m.CanRead,
			CanWrite: m.CanWrite,
		}
	}
	return perms, nil
}

func (r *RoleRepository) SetPermissions(ctx context.Context, roleID uuid.UUID, perms []domain.RolePermission) error {
	db := GetDB(ctx, r.db)
	if err := db.Where("role_id = ?", roleID).Delete(&RolePermission{}).Error; err != nil {
		return err
	}
	for _, p := range perms {
		m := RolePermission{
			RoleID:   roleID,
			Module:   p.Module,
			CanRead:  p.CanRead,
			CanWrite: p.CanWrite,
		}
		if err := db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRepository) GetEnvironmentIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	var models []RoleEnvironment
	if err := GetDB(ctx, r.db).Where("role_id = ?", roleID).Find(&models).Error; err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(models))
	for i, m := range models {
		ids[i] = m.EnvironmentID
	}
	return ids, nil
}

func (r *RoleRepository) SetEnvironments(ctx context.Context, roleID uuid.UUID, envIDs []uuid.UUID) error {
	db := GetDB(ctx, r.db)
	if err := db.Where("role_id = ?", roleID).Delete(&RoleEnvironment{}).Error; err != nil {
		return err
	}
	for _, eid := range envIDs {
		m := RoleEnvironment{
			RoleID:        roleID,
			EnvironmentID: eid,
		}
		if err := db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRepository) DeleteEnvironmentByEnvID(ctx context.Context, envID uuid.UUID) error {
	return GetDB(ctx, r.db).Where("environment_id = ?", envID).Delete(&RoleEnvironment{}).Error
}

func roleToDomain(m *Role) *domain.Role {
	return &domain.Role{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func roleToModel(d *domain.Role) *Role {
	return &Role{
		ID:          d.ID,
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
