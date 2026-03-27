package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type UserService struct {
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
}

func NewUserService(userRepo domain.UserRepository, roleRepo domain.RoleRepository) *UserService {
	return &UserService{userRepo: userRepo, roleRepo: roleRepo}
}

func (s *UserService) Create(ctx context.Context, username, password string, roleID uuid.UUID) (*domain.User, error) {
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return nil, errors.New("username already exists")
	}
	// 验证角色存在
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return nil, errors.New("role not found")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		Username:     username,
		PasswordHash: string(hash),
		IsSuper:      false,
		RoleID:       &roleID,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]domain.User, int64, error) {
	users, total, err := s.userRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	// 填充角色名称
	for i := range users {
		if users[i].RoleID != nil {
			if role, err := s.roleRepo.GetByID(ctx, *users[i].RoleID); err == nil {
				users[i].RoleName = role.Name
			}
		}
	}
	return users, total, nil
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, roleID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user.IsSuper {
		return errors.New("cannot modify the super admin")
	}
	// 验证角色存在
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return errors.New("role not found")
	}
	user.RoleID = &roleID
	return s.userRepo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user.IsSuper {
		return errors.New("cannot delete the super admin")
	}
	return s.userRepo.Delete(ctx, id)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.RoleID != nil {
		if role, err := s.roleRepo.GetByID(ctx, *user.RoleID); err == nil {
			user.RoleName = role.Name
		}
	}
	return user, nil
}

// TransferSuper 转移超级管理员权限
func (s *UserService) TransferSuper(ctx context.Context, currentUserID, targetUserID uuid.UUID, roleIDForOld uuid.UUID) error {
	current, err := s.userRepo.GetByID(ctx, currentUserID)
	if err != nil {
		return err
	}
	if !current.IsSuper {
		return errors.New("only super admin can transfer")
	}
	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}
	if target.IsSuper {
		return errors.New("target is already super admin")
	}

	// 旧超管降级
	current.IsSuper = false
	current.RoleID = &roleIDForOld
	if err := s.userRepo.Update(ctx, current); err != nil {
		return err
	}

	// 新超管升级
	target.IsSuper = true
	target.RoleID = nil
	return s.userRepo.Update(ctx, target)
}
