package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type configAuthorizationEnvironmentRepository struct {
	environment *domain.Environment
}

func (r *configAuthorizationEnvironmentRepository) Create(context.Context, *domain.Environment) error {
	panic("not used")
}
func (r *configAuthorizationEnvironmentRepository) GetByID(context.Context, uuid.UUID) (*domain.Environment, error) {
	panic("not used")
}
func (r *configAuthorizationEnvironmentRepository) GetByName(context.Context, string) (*domain.Environment, error) {
	return r.environment, nil
}
func (r *configAuthorizationEnvironmentRepository) List(context.Context) ([]domain.Environment, error) {
	panic("not used")
}
func (r *configAuthorizationEnvironmentRepository) Update(context.Context, *domain.Environment) error {
	panic("not used")
}
func (r *configAuthorizationEnvironmentRepository) Delete(context.Context, uuid.UUID) error {
	panic("not used")
}

func TestConfigServiceWritesValidateBeforeAccessingDependencies(t *testing.T) {
	svc := &ConfigService{}
	writes := map[string]func() error{
		"create": func() error {
			return svc.Create(context.Background(), "dev", "app.yaml", "items: [one, two", "test", uuid.Nil)
		},
		"update": func() error {
			return svc.Update(context.Background(), "dev", "app.yaml", "items: [one, two", "test", uuid.Nil)
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			err := write()
			var validationErr *ConfigValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("write error = %v, want *ConfigValidationError", err)
			}
		})
	}
}

func TestConfigServiceImportDryRunValidatesConfigValues(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), Name: "dev"}
	svc := newConfigServiceForAuthorizationTest(env)
	data := []byte(`{"good.yaml":"name: app\\n","bad.json":"{\\\"broken\\\":"}`)
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{Unrestricted: true})
	result, err := svc.Import(ctx, "dev", data, true, uuid.Nil)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Total != 2 || result.Success != 1 || len(result.Failed) != 1 {
		t.Fatalf("Import() result = %+v, want total=2 success=1 failed=1", result)
	}
	if !strings.Contains(result.Failed[0], "bad.json") ||
		!strings.Contains(result.Failed[0], "invalid JSON config") {
		t.Fatalf("Import() failure = %q, want key and validation reason", result.Failed[0])
	}
}

func TestConfigServiceListRejectsUnauthorizedEnvironment(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), Name: "prod"}
	svc := newConfigServiceForAuthorizationTest(env)
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{uuid.New()}})

	_, err := svc.List(ctx, "prod", "")

	if !errors.Is(err, domain.ErrEnvironmentForbidden) {
		t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
	}
}

func newConfigServiceForAuthorizationTest(env *domain.Environment) *ConfigService {
	return &ConfigService{envRepo: &configAuthorizationEnvironmentRepository{environment: env}}
}
