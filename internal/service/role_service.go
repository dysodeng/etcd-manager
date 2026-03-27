package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type RoleService struct {
	roleRepo domain.RoleRepository
	userRepo domain.UserRepository
}

func NewRoleService(roleRepo domain.RoleRepository, userRepo domain.UserRepository) *RoleService {
	return &RoleService{roleRepo: roleRepo, userRepo: userRepo}
}

type RoleCreateRequest struct {
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Permissions    []domain.RolePermission   `json:"permissions"`
	EnvironmentIDs []uuid.UUID              `json:"environment_ids"`
}

func (s *RoleService) Create(ctx context.Context, req *RoleCreateRequest) (*domain.Role, error) {
	if _, err := s.roleRepo.GetByName(ctx, req.Name); err == nil {
		return nil, errors.New("role name already exists")
	}
	role := &domain.Role{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	if err := s.roleRepo.SetPermissions(ctx, role.ID, req.Permissions); err != nil {
		return nil, err
	}
	if err := s.roleRepo.SetEnvironments(ctx, role.ID, req.EnvironmentIDs); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RoleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	perms, _ := s.roleRepo.GetPermissions(ctx, id)
	role.Permissions = perms
	envIDs, _ := s.roleRepo.GetEnvironmentIDs(ctx, id)
	role.Environments = envIDs
	userCount, _ := s.userRepo.CountByRoleID(ctx, id)
	role.UserCount = userCount
	return role, nil
}

func (s *RoleService) List(ctx context.Context, page, pageSize int) ([]domain.Role, int64, error) {
	roles, total, err := s.roleRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range roles {
		count, _ := s.userRepo.CountByRoleID(ctx, roles[i].ID)
		roles[i].UserCount = count
	}
	return roles, total, nil
}

func (s *RoleService) Update(ctx context.Context, id uuid.UUID, req *RoleCreateRequest) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return errors.New("role not found")
	}
	// 检查名称唯一性
	if existing, err := s.roleRepo.GetByName(ctx, req.Name); err == nil && existing.ID != id {
		return errors.New("role name already exists")
	}
	role.Name = req.Name
	role.Description = req.Description
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return err
	}
	if err := s.roleRepo.SetPermissions(ctx, id, req.Permissions); err != nil {
		return err
	}
	return s.roleRepo.SetEnvironments(ctx, id, req.EnvironmentIDs)
}

func (s *RoleService) Delete(ctx context.Context, id uuid.UUID) error {
	count, err := s.userRepo.CountByRoleID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("cannot delete role with assigned users")
	}
	return s.roleRepo.Delete(ctx, id)
}
