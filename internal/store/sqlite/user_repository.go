package sqlite

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return GetDB(ctx, r.db).Create(user).Error
}
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	return &u, GetDB(ctx, r.db).First(&u, id).Error
}
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	return &u, GetDB(ctx, r.db).Where("username = ?", username).First(&u).Error
}
func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	db := GetDB(ctx, r.db).Model(&model.User{})
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id ASC").Find(&users).Error
	return users, total, err
}
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return GetDB(ctx, r.db).Save(user).Error
}
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	return GetDB(ctx, r.db).Delete(&model.User{}, id).Error
}
