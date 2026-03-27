package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/middleware"
)

type AuthService struct {
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
	jwtSecret string
	expireH   int
}

func NewAuthService(userRepo domain.UserRepository, roleRepo domain.RoleRepository, jwtSecret string, expireH int) *AuthService {
	return &AuthService{userRepo: userRepo, roleRepo: roleRepo, jwtSecret: jwtSecret, expireH: expireH}
}

type LoginResult struct {
	Token    string      `json:"token"`
	UserID   string      `json:"user_id"`
	Username string      `json:"username"`
	IsSuper  bool        `json:"is_super"`
	Role     *RoleDetail `json:"role"`
}

type RoleDetail struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Permissions    []domain.RolePermission `json:"permissions"`
	EnvironmentIDs []string              `json:"environment_ids"`
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	roleIDStr := ""
	if user.RoleID != nil {
		roleIDStr = user.RoleID.String()
	}

	claims := &middleware.Claims{
		UserID:   user.ID.String(),
		Username: user.Username,
		IsSuper:  user.IsSuper,
		RoleID:   roleIDStr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.expireH) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	result := &LoginResult{
		Token:    tokenStr,
		UserID:   user.ID.String(),
		Username: user.Username,
		IsSuper:  user.IsSuper,
	}

	// 获取角色详情
	if user.RoleID != nil {
		roleDetail, err := s.buildRoleDetail(ctx, *user.RoleID)
		if err == nil {
			result.Role = roleDetail
		}
	}

	return result, nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*LoginResult, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := &LoginResult{
		UserID:   user.ID.String(),
		Username: user.Username,
		IsSuper:  user.IsSuper,
	}
	if user.RoleID != nil {
		roleDetail, err := s.buildRoleDetail(ctx, *user.RoleID)
		if err == nil {
			result.Role = roleDetail
		}
	}
	return result, nil
}

func (s *AuthService) buildRoleDetail(ctx context.Context, roleID uuid.UUID) (*RoleDetail, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	perms, _ := s.roleRepo.GetPermissions(ctx, roleID)
	envIDs, _ := s.roleRepo.GetEnvironmentIDs(ctx, roleID)

	envIDStrs := make([]string, len(envIDs))
	for i, id := range envIDs {
		envIDStrs[i] = id.String()
	}

	return &RoleDetail{
		ID:             role.ID.String(),
		Name:           role.Name,
		Permissions:    perms,
		EnvironmentIDs: envIDStrs,
	}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPwd, newPwd string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPwd)); err != nil {
		return errors.New("old password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	return s.userRepo.Update(ctx, user)
}
