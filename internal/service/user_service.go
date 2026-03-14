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
}

func NewUserService(userRepo domain.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) Create(ctx context.Context, username, password, role string) (*domain.User, error) {
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return nil, errors.New("username already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]domain.User, int64, error) {
	return s.userRepo.List(ctx, page, pageSize)
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, role string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user.Username == "admin" {
		return errors.New("cannot modify the super admin")
	}
	user.Role = role
	return s.userRepo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user.Username == "admin" {
		return errors.New("cannot delete the super admin")
	}
	return s.userRepo.Delete(ctx, id)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
