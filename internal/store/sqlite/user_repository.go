package sqlite

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dysodeng/config-center/internal/domain"
	"github.com/dysodeng/config-center/internal/model"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	user.ID = uuid.Must(uuid.NewV7())
	m := userToModel(user)
	if err := GetDB(ctx, r.db).Create(m).Error; err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var m model.User
	if err := GetDB(ctx, r.db).Where("id = ?", id.String()).First(&m).Error; err != nil {
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var m model.User
	if err := GetDB(ctx, r.db).Where("username = ?", username).First(&m).Error; err != nil {
		return nil, err
	}
	return userToDomain(&m), nil
}

func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]domain.User, int64, error) {
	var models []model.User
	var total int64
	db := GetDB(ctx, r.db).Model(&model.User{})
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
	return GetDB(ctx, r.db).Where("id = ?", id.String()).Delete(&model.User{}).Error
}

func userToDomain(m *model.User) *domain.User {
	return &domain.User{
		ID:           uuid.MustParse(m.ID),
		Username:     m.Username,
		PasswordHash: m.PasswordHash,
		Role:         m.Role,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func userToModel(d *domain.User) *model.User {
	return &model.User{
		ID:           d.ID.String(),
		Username:     d.Username,
		PasswordHash: d.PasswordHash,
		Role:         d.Role,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}
