package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type fakeEnvironmentRoleRepository struct {
	environmentIDs []uuid.UUID
}

func (r *fakeEnvironmentRoleRepository) Create(context.Context, *domain.Role) error {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) GetByID(context.Context, uuid.UUID) (*domain.Role, error) {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) GetByName(context.Context, string) (*domain.Role, error) {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) List(context.Context, int, int) ([]domain.Role, int64, error) {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) Update(context.Context, *domain.Role) error {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) Delete(context.Context, uuid.UUID) error { panic("not used") }
func (r *fakeEnvironmentRoleRepository) GetPermissions(context.Context, uuid.UUID) ([]domain.RolePermission, error) {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) SetPermissions(context.Context, uuid.UUID, []domain.RolePermission) error {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) GetEnvironmentIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return r.environmentIDs, nil
}
func (r *fakeEnvironmentRoleRepository) SetEnvironments(context.Context, uuid.UUID, []uuid.UUID) error {
	panic("not used")
}
func (r *fakeEnvironmentRoleRepository) DeleteEnvironmentByEnvID(context.Context, uuid.UUID) error {
	panic("not used")
}

func TestFilterEnvironmentsInjectsRequestScope(t *testing.T) {
	allowedID := uuid.New()
	roleID := uuid.New()
	repo := &fakeEnvironmentRoleRepository{environmentIDs: []uuid.UUID{allowedID}}
	c := newEnvironmentFilterContext(t)
	c.Set("role_id", roleID.String())
	c.Set("is_super", false)

	FilterEnvironments(repo)(c)

	if err := domain.RequireEnvironmentAccess(c.Request.Context(), allowedID); err != nil {
		t.Fatalf("scope error = %v", err)
	}
}

func TestFilterEnvironmentsInjectsUnrestrictedScopeForSuper(t *testing.T) {
	c := newEnvironmentFilterContext(t)
	c.Set("is_super", true)

	FilterEnvironments(&fakeEnvironmentRoleRepository{})(c)

	if err := domain.RequireEnvironmentAccess(c.Request.Context(), uuid.New()); err != nil {
		t.Fatalf("scope error = %v", err)
	}
}

func newEnvironmentFilterContext(t *testing.T) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c
}
