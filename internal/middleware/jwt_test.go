package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type fakeJWTUserRepository struct {
	users map[uuid.UUID]*domain.User
	err   error
}

func (r *fakeJWTUserRepository) Create(context.Context, *domain.User) error { panic("not used") }
func (r *fakeJWTUserRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	copy := *user
	return &copy, nil
}
func (r *fakeJWTUserRepository) GetByUsername(context.Context, string) (*domain.User, error) {
	panic("not used")
}
func (r *fakeJWTUserRepository) List(context.Context, int, int) ([]domain.User, int64, error) {
	panic("not used")
}
func (r *fakeJWTUserRepository) Update(context.Context, *domain.User) error { panic("not used") }
func (r *fakeJWTUserRepository) Delete(context.Context, uuid.UUID) error    { panic("not used") }
func (r *fakeJWTUserRepository) CountByRoleID(context.Context, uuid.UUID) (int64, error) {
	panic("not used")
}
func (r *fakeJWTUserRepository) GetSuperAdmin(context.Context) (*domain.User, error) {
	panic("not used")
}

func TestJWTAuthReloadsCurrentUser(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	repo := &fakeJWTUserRepository{users: map[uuid.UUID]*domain.User{
		userID: {ID: userID, Username: "current", IsSuper: true, RoleID: &roleID},
	}}
	token := signJWTTestToken(t, jwt.SigningMethodHS256, "secret", Claims{UserID: userID.String()})

	status, values := runJWTTestRequest(t, token, repo)

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if values["username"] != "current" {
		t.Fatalf("username = %v, want current", values["username"])
	}
	if values["is_super"] != true {
		t.Fatalf("is_super = %v, want true", values["is_super"])
	}
	if values["role_id"] != roleID.String() {
		t.Fatalf("role_id = %v, want %s", values["role_id"], roleID)
	}
}

func TestJWTAuthRejectsDeletedUser(t *testing.T) {
	userID := uuid.New()
	repo := &fakeJWTUserRepository{users: map[uuid.UUID]*domain.User{}}
	token := signJWTTestToken(t, jwt.SigningMethodHS256, "secret", Claims{UserID: userID.String()})

	status, _ := runJWTTestRequest(t, token, repo)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestJWTAuthRejectsNonHS256Token(t *testing.T) {
	userID := uuid.New()
	repo := &fakeJWTUserRepository{users: map[uuid.UUID]*domain.User{
		userID: {ID: userID},
	}}
	token := signJWTTestToken(t, jwt.SigningMethodHS384, "secret", Claims{UserID: userID.String()})

	status, _ := runJWTTestRequest(t, token, repo)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestJWTAuthReflectsSuperTransfer(t *testing.T) {
	oldID, newID := uuid.New(), uuid.New()
	repo := &fakeJWTUserRepository{users: map[uuid.UUID]*domain.User{
		oldID: {ID: oldID, Username: "old", IsSuper: true},
		newID: {ID: newID, Username: "new", IsSuper: false},
	}}
	oldToken := signJWTTestToken(t, jwt.SigningMethodHS256, "secret", Claims{UserID: oldID.String()})
	newToken := signJWTTestToken(t, jwt.SigningMethodHS256, "secret", Claims{UserID: newID.String()})

	_, oldBefore := runJWTTestRequest(t, oldToken, repo)
	_, newBefore := runJWTTestRequest(t, newToken, repo)
	if oldBefore["is_super"] != true || newBefore["is_super"] != false {
		t.Fatalf("before transfer: old=%v new=%v", oldBefore["is_super"], newBefore["is_super"])
	}

	repo.users[oldID].IsSuper = false
	repo.users[newID].IsSuper = true

	_, oldAfter := runJWTTestRequest(t, oldToken, repo)
	_, newAfter := runJWTTestRequest(t, newToken, repo)
	if oldAfter["is_super"] != false || newAfter["is_super"] != true {
		t.Fatalf("after transfer: old=%v new=%v", oldAfter["is_super"], newAfter["is_super"])
	}
}

func signJWTTestToken(t *testing.T, method jwt.SigningMethod, secret string, claims Claims) string {
	t.Helper()
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func runJWTTestRequest(t *testing.T, token string, repo domain.UserRepository) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	values := make(map[string]any)
	router.Use(JWTAuth("secret", repo))
	router.GET("/", func(c *gin.Context) {
		for _, key := range []string{"user_id", "username", "is_super", "role_id"} {
			value, _ := c.Get(key)
			values[key] = value
		}
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder.Code, values
}
