package sqlite

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type RoleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) *RoleRepository { return &RoleRepository{db: db} }

func (r *RoleRepository) Create(ctx context.Context, role *domain.Role) error {
	role.ID = uuid.Must(uuid.NewV7())
	return GetDB(ctx, r.db).Create(roleToModel(role)).Error
}

func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	var m Role
	if err := GetDB(ctx, r.db).Where("id = ?", id.String()).First(&m).Error; err != nil {
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
	db.Where("role_id = ?", id.String()).Delete(&RolePermission{})
	db.Where("role_id = ?", id.String()).Delete(&RoleEnvironment{})
	return db.Where("id = ?", id.String()).Delete(&Role{}).Error
}

func (r *RoleRepository) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]domain.RolePermission, error) {
	var models []RolePermission
	if err := GetDB(ctx, r.db).Where("role_id = ?", roleID.String()).Find(&models).Error; err != nil {
		return nil, err
	}
	perms := make([]domain.RolePermission, len(models))
	for i, m := range models {
		perms[i] = domain.RolePermission{
			ID:       uuid.MustParse(m.ID),
			RoleID:   uuid.MustParse(m.RoleID),
			Module:   m.Module,
			CanRead:  m.CanRead,
			CanWrite: m.CanWrite,
		}
	}
	return perms, nil
}

func (r *RoleRepository) SetPermissions(ctx context.Context, roleID uuid.UUID, perms []domain.RolePermission) error {
	db := GetDB(ctx, r.db)
	if err := db.Where("role_id = ?", roleID.String()).Delete(&RolePermission{}).Error; err != nil {
		return err
	}
	for _, p := range perms {
		m := RolePermission{
			ID:       uuid.Must(uuid.NewV7()).String(),
			RoleID:   roleID.String(),
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
	if err := GetDB(ctx, r.db).Where("role_id = ?", roleID.String()).Find(&models).Error; err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(models))
	for i, m := range models {
		ids[i] = uuid.MustParse(m.EnvironmentID)
	}
	return ids, nil
}

func (r *RoleRepository) SetEnvironments(ctx context.Context, roleID uuid.UUID, envIDs []uuid.UUID) error {
	db := GetDB(ctx, r.db)
	if err := db.Where("role_id = ?", roleID.String()).Delete(&RoleEnvironment{}).Error; err != nil {
		return err
	}
	for _, eid := range envIDs {
		m := RoleEnvironment{
			ID:            uuid.Must(uuid.NewV7()).String(),
			RoleID:        roleID.String(),
			EnvironmentID: eid.String(),
		}
		if err := db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRepository) DeleteEnvironmentByEnvID(ctx context.Context, envID uuid.UUID) error {
	return GetDB(ctx, r.db).Where("environment_id = ?", envID.String()).Delete(&RoleEnvironment{}).Error
}

func roleToDomain(m *Role) *domain.Role {
	return &domain.Role{
		ID:          uuid.MustParse(m.ID),
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func roleToModel(d *domain.Role) *Role {
	return &Role{
		ID:          d.ID.String(),
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
