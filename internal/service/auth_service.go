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
	userRepo  domain.UserRepository
	jwtSecret string
	expireH   int
}

func NewAuthService(userRepo domain.UserRepository, jwtSecret string, expireH int) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret, expireH: expireH}
}

type LoginResult struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid username or password")
	}
	claims := &middleware.Claims{
		UserID:   user.ID.String(),
		Username: user.Username,
		Role:     user.Role,
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
	return &LoginResult{
		Token:    tokenStr,
		UserID:   user.ID.String(),
		Username: user.Username,
		Role:     user.Role,
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
