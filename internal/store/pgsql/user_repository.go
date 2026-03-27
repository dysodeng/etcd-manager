package pgsql

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	m := userToModel(user)
	if err := GetDB(ctx, r.db).Create(m).Error; err != nil {
		return err
	}
	user.ID = m.ID
	user.CreatedAt = m.CreatedAt
	user.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var m User
	if err := GetDB(ctx, r.db).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var m User
	if err := GetDB(ctx, r.db).Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]domain.User, int64, error) {
	var models []User
	var total int64
	db := GetDB(ctx, r.db).Model(&User{})
	db.Count(&total)
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id ASC").Find(&models).Error; err != nil {
		return nil, 0, err
	}
	users := make([]domain.User, len(models))
	for i := range models {
		users[i] = *userToDomain(&models[i])
	}
	return users, total, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	return GetDB(ctx, r.db).Save(userToModel(user)).Error
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return GetDB(ctx, r.db).Where("id = ?", id).Delete(&User{}).Error
}

func (r *UserRepository) CountByRoleID(ctx context.Context, roleID uuid.UUID) (int64, error) {
	var count int64
	err := GetDB(ctx, r.db).Model(&User{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}

func (r *UserRepository) GetSuperAdmin(ctx context.Context) (*domain.User, error) {
	var m User
	if err := GetDB(ctx, r.db).Where("is_super = ?", true).First(&m).Error; err != nil {
		return nil, err
	}
	return userToDomain(&m), nil
}

func userToDomain(m *User) *domain.User {
	u := &domain.User{
		ID:           m.ID,
		Username:     m.Username,
		PasswordHash: m.PasswordHash,
		IsSuper:      m.IsSuper,
		RoleID:       m.RoleID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	return u
}

func userToModel(d *domain.User) *User {
	return &User{
		ID:           d.ID,
		Username:     d.Username,
		PasswordHash: d.PasswordHash,
		IsSuper:      d.IsSuper,
		RoleID:       d.RoleID,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}
