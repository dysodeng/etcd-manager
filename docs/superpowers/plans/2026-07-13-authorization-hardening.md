# Authorization Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce current-user JWT authorization and environment isolation for configuration, gateway, and gRPC operations.

**Architecture:** JWT middleware verifies HS256 then reloads the user from the repository. Environment scope is injected into request context by middleware and enforced in services, while generic KV/Watch retain global-key semantics. Super-admin transfer uses the existing transaction abstraction.

**Tech Stack:** Go 1.25, Gin, jwt/v5, GORM repository abstractions, React 18, TypeScript, Go table-driven tests.

---

## File Structure

- Create `internal/domain/access.go`: environment scope context and typed authorization error.
- Create `internal/domain/access_test.go`: default-deny, restricted, and unrestricted scope tests.
- Modify `internal/middleware/jwt.go`: HS256 restriction and current-user reload.
- Create `internal/middleware/jwt_test.go`: token algorithm, stale claim, and deleted-user tests.
- Modify `internal/middleware/role.go`: inject environment scope into request context.
- Create `internal/middleware/role_test.go`: environment-scope middleware tests.
- Modify `internal/handler/router.go`, `cmd/server/main.go`: wire user repository and remove environment middleware from KV/Watch.
- Modify `internal/service/config_service.go`: require environment scope for every config entry point.
- Modify `internal/handler/config_center.go`: map authorization errors to forbidden.
- Modify gateway/gRPC handlers and services: accept environment, derive prefixes server-side, reject foreign keys.
- Create `internal/service/service_registry_store.go`: narrow etcd interface used by gateway/gRPC services and their fakes.
- Modify gateway/gRPC frontend APIs and pages: send `env` instead of prefix.
- Modify `internal/service/user_service.go`, `cmd/server/main.go`: transactional super transfer.
- Add focused tests beside each modified package.

### Task 1: Environment Access Scope

**Files:**
- Create: `internal/domain/access.go`
- Create: `internal/domain/access_test.go`

- [ ] **Step 1: Write failing scope tests**

~~~go
package domain

import (
    "context"
    "errors"
    "testing"

    "github.com/google/uuid"
)

func TestRequireEnvironmentAccess(t *testing.T) {
    allowedID := uuid.New()
    deniedID := uuid.New()
    tests := []struct {
        name string
        ctx context.Context
        envID uuid.UUID
        wantErr bool
    }{
        {name: "missing scope defaults to deny", ctx: context.Background(), envID: allowedID, wantErr: true},
        {name: "allowed environment", ctx: WithEnvironmentScope(context.Background(), EnvironmentScope{AllowedIDs: []uuid.UUID{allowedID}}), envID: allowedID},
        {name: "other environment denied", ctx: WithEnvironmentScope(context.Background(), EnvironmentScope{AllowedIDs: []uuid.UUID{allowedID}}), envID: deniedID, wantErr: true},
        {name: "unrestricted", ctx: WithEnvironmentScope(context.Background(), EnvironmentScope{Unrestricted: true}), envID: deniedID},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := RequireEnvironmentAccess(tt.ctx, tt.envID)
            if tt.wantErr && !errors.Is(err, ErrEnvironmentForbidden) {
                t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
            }
            if !tt.wantErr && err != nil {
                t.Fatalf("error = %v, want nil", err)
            }
        })
    }
}
~~~

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/domain -run '^TestRequireEnvironmentAccess$'`

Expected: build fails because the scope API is undefined.

- [ ] **Step 3: Implement the scope**

~~~go
package domain

import (
    "context"
    "errors"

    "github.com/google/uuid"
)

var ErrEnvironmentForbidden = errors.New("environment access denied")

type EnvironmentScope struct {
    Unrestricted bool
    AllowedIDs []uuid.UUID
}

type environmentScopeKey struct{}

func WithEnvironmentScope(ctx context.Context, scope EnvironmentScope) context.Context {
    return context.WithValue(ctx, environmentScopeKey{}, scope)
}

func RequireEnvironmentAccess(ctx context.Context, environmentID uuid.UUID) error {
    scope, ok := ctx.Value(environmentScopeKey{}).(EnvironmentScope)
    if !ok {
        return ErrEnvironmentForbidden
    }
    if scope.Unrestricted {
        return nil
    }
    for _, id := range scope.AllowedIDs {
        if id == environmentID {
            return nil
        }
    }
    return ErrEnvironmentForbidden
}
~~~

- [ ] **Step 4: Verify GREEN and commit**

Run: `gofmt -w internal/domain/access.go internal/domain/access_test.go && go test ./internal/domain`

Expected: PASS.

~~~bash
git add internal/domain/access.go internal/domain/access_test.go
git commit -m "feat: add environment access scope"
~~~

### Task 2: JWT Current-User Authentication

**Files:**
- Modify: `internal/middleware/jwt.go`
- Create: `internal/middleware/jwt_test.go`
- Modify: `internal/service/auth_service.go`
- Modify: `internal/handler/router.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write failing middleware tests**

Create `internal/middleware/jwt_test.go` in package `middleware`. Add a `fakeUserRepository` implementing every `domain.UserRepository` method (unused methods may panic), plus `signTestToken` and `runJWTMiddleware` helpers, then cover:

~~~go
func TestJWTAuthReloadsCurrentUser(t *testing.T) {
    userID := uuid.New()
    currentRole := uuid.New()
    repo := &fakeUserRepository{user: &domain.User{ID: userID, Username: "current", RoleID: &currentRole}}
    token := signTestToken(t, jwt.SigningMethodHS256, "secret", Claims{
        UserID: userID.String(),
        Username: "stale",
        IsSuper: true,
    })
    ctx, recorder := runJWTMiddleware(token, repo)
    if recorder.Code != http.StatusOK {
        t.Fatalf("status = %d", recorder.Code)
    }
    if got, _ := ctx.Get("is_super"); got != false {
        t.Fatalf("is_super = %v, want current database value false", got)
    }
    if got, _ := ctx.Get("role_id"); got != currentRole.String() {
        t.Fatalf("role_id = %v", got)
    }
}

func TestJWTAuthRejectsDeletedUser(t *testing.T) {
    repo := &fakeUserRepository{getErr: gorm.ErrRecordNotFound}
    token := signValidTestToken(t, uuid.New(), "secret")
    _, recorder := runJWTMiddleware(token, repo)
    if recorder.Code != http.StatusUnauthorized {
        t.Fatalf("status = %d, want 401", recorder.Code)
    }
}

func TestJWTAuthRejectsNonHS256Token(t *testing.T) {
    repo := &fakeUserRepository{}
    token := signTestToken(t, jwt.SigningMethodHS384, "secret", Claims{UserID: uuid.NewString()})
    _, recorder := runJWTMiddleware(token, repo)
    if recorder.Code != http.StatusUnauthorized {
        t.Fatalf("status = %d, want 401", recorder.Code)
    }
}
~~~

Also add `TestJWTAuthReflectsSuperTransfer`: authenticate the same two HS256 tokens before and after swapping `IsSuper` in the fake repository, and assert the old super loses and the target gains super status without issuing new tokens.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/middleware -run '^TestJWTAuth'`

Expected: tests fail because `JWTAuth` does not accept a repository and trusts stale claims.

- [ ] **Step 3: Implement current-user reload**

Reduce the signed authorization identity to the immutable user ID:

~~~go
type Claims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
}
~~~

Change the middleware signature and parsing:

~~~go
func JWTAuth(secret string, userRepo domain.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := &Claims{}
        token, err := jwt.ParseWithClaims(
            extractToken(c),
            claims,
            func(_ *jwt.Token) (any, error) { return []byte(secret), nil },
            jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
        )
        if err != nil || !token.Valid {
            response.FailUnauthorized(c, "invalid or expired token")
            c.Abort()
            return
        }
        userID, err := uuid.Parse(claims.UserID)
        if err != nil {
            response.FailUnauthorized(c, "invalid user identity")
            c.Abort()
            return
        }
        user, err := userRepo.GetByID(c.Request.Context(), userID)
        if err != nil {
            response.FailUnauthorized(c, "user not found")
            c.Abort()
            return
        }
        roleID := ""
        if user.RoleID != nil {
            roleID = user.RoleID.String()
        }
        c.Set("user_id", user.ID.String())
        c.Set("username", user.Username)
        c.Set("is_super", user.IsSuper)
        c.Set("role_id", roleID)
        c.Next()
    }
}
~~~

Remove `Username`, `IsSuper`, and `RoleID` from both `Claims` and the login token payload. Pass `userRepo` through `RegisterRoutes` and `main`.

- [ ] **Step 4: Verify GREEN and commit**

Run: `gofmt -w internal/middleware/jwt.go internal/middleware/jwt_test.go internal/service/auth_service.go internal/handler/router.go cmd/server/main.go && go test ./internal/middleware ./internal/service ./internal/handler`

Expected: PASS.

~~~bash
git add internal/middleware/jwt.go internal/middleware/jwt_test.go internal/service/auth_service.go internal/handler/router.go cmd/server/main.go
git commit -m "fix: reload current user for JWT authorization"
~~~

### Task 3: Inject and Enforce Environment Scope

**Files:**
- Modify: `internal/middleware/role.go`
- Create: `internal/middleware/role_test.go`
- Modify: `internal/handler/router.go`
- Modify: `internal/service/config_service.go`
- Modify: `internal/service/config_service_test.go`
- Modify: `internal/handler/config_center.go`

- [ ] **Step 1: Write failing middleware and config tests**

~~~go
func TestFilterEnvironmentsInjectsRequestScope(t *testing.T) {
    allowedID := uuid.New()
    roleID := uuid.New()
    repo := &fakeRoleRepository{environmentIDs: []uuid.UUID{allowedID}}
    ctx := newRoleContext(roleID.String(), false)
    FilterEnvironments(repo)(ctx)
    if err := domain.RequireEnvironmentAccess(ctx.Request.Context(), allowedID); err != nil {
        t.Fatalf("scope error = %v", err)
    }
}

func TestConfigServiceListRejectsUnauthorizedEnvironment(t *testing.T) {
    env := domain.Environment{ID: uuid.New(), Name: "prod"}
    svc := newConfigServiceForAuthorizationTest(env)
    ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{uuid.New()}})
    _, err := svc.List(ctx, "prod", "")
    if !errors.Is(err, domain.ErrEnvironmentForbidden) {
        t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
    }
}
~~~

In `internal/service/config_service_test.go`, define `newConfigServiceForAuthorizationTest` with a fake environment repository returning the supplied environment; the etcd and revision fields may stay nil because the denied/dry-run paths must return before using them. Update the existing import dry-run test to use this fixture and `domain.WithEnvironmentScope(...Unrestricted: true)`; authorization must happen before dry-run validation, while the test must still avoid etcd/revision writes.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/middleware ./internal/service -run 'Environment|Unauthorized'`

Expected: scope is absent from request context and config service does not reject the environment.

- [ ] **Step 3: Inject scope in middleware**

For super users:

~~~go
c.Request = c.Request.WithContext(domain.WithEnvironmentScope(
    c.Request.Context(),
    domain.EnvironmentScope{Unrestricted: true},
))
~~~

For role users, inject `EnvironmentScope{AllowedIDs: envIDs}` and keep `allowed_env_ids` for the environment-list response.

- [ ] **Step 4: Enforce scope in every config service entry point**

Centralize lookup and authorization:

~~~go
func (s *ConfigService) resolveAuthorizedEnvironment(ctx context.Context, envName string) (*domain.Environment, error) {
    env, err := s.envRepo.GetByName(ctx, envName)
    if err != nil {
        return nil, fmt.Errorf("environment not found: %s", envName)
    }
    if err := domain.RequireEnvironmentAccess(ctx, env.ID); err != nil {
        return nil, err
    }
    return env, nil
}
~~~

Use this helper in `List`, `Create`, `Update`, `Delete`, `Revisions`, `Rollback`, and `Import`. `Rollback` must resolve/authorize before fetching the revision. `Export` continues to delegate to `List`, so it inherits the same check without a duplicate lookup. Resolve and authorize before dry-run import validation.

Extend `configWriteErrorCode`:

~~~go
if errors.Is(err, domain.ErrEnvironmentForbidden) {
    return CodeForbidden
}
~~~

Ensure config list, create, update, delete, revisions, rollback, export, and import all check this classifier for `ErrEnvironmentForbidden`. Preserve their current non-authorization mappings (especially `CodeImportFormat` and `CodeImportPartial`). Add handler table tests showing `ErrEnvironmentForbidden` maps to `CodeForbidden` for both read and write paths.

- [ ] **Step 5: Remove meaningless KV/Watch environment filtering**

Use:

~~~go
kv := auth.Group("/kv", middleware.RequirePermission("kv", roleRepo))
auth.GET("/watch", middleware.RequirePermission("kv", roleRepo), h.Watch.Watch)
~~~

- [ ] **Step 6: Verify GREEN and commit**

Run: `gofmt -w internal/middleware/role.go internal/middleware/role_test.go internal/handler/router.go internal/service/config_service.go internal/service/config_service_test.go internal/handler/config_center.go && go test ./internal/middleware ./internal/service ./internal/handler`

Expected: PASS.

~~~bash
git add internal/middleware/role.go internal/middleware/role_test.go internal/handler/router.go internal/service/config_service.go internal/service/config_service_test.go internal/handler/config_center.go
git commit -m "fix: enforce configuration environment access"
~~~

### Task 4: Gateway and gRPC Environment Boundaries

**Files:**
- Modify: `internal/handler/gateway.go`
- Modify: `internal/handler/grpc.go`
- Modify: `internal/service/gateway_service.go`
- Modify: `internal/service/grpc_service.go`
- Create: `internal/service/service_registry_store.go`
- Create: `internal/service/service_registry_access_test.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write failing service-boundary tests**

~~~go
func TestGatewayRejectsKeyOutsideEnvironmentPrefix(t *testing.T) {
    env := &domain.Environment{ID: uuid.New(), KeyPrefix: "/prod/", GatewayPrefix: "gw-services/"}
    ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{env.ID}})
    store := &fakeServiceRegistryStore{}
    svc := NewGatewayService(store)
    err := svc.UpdateInstanceStatus(ctx, env, "/other/gw-services/app/1", "down")
    if !errors.Is(err, domain.ErrEnvironmentForbidden) {
        t.Fatalf("error = %v", err)
    }
    if store.getCalls != 0 {
        t.Fatal("etcd must not be called for foreign key")
    }
}
~~~

Add the same test for gRPC and list operations with unauthorized environments.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service -run 'Gateway|Grpc.*Environment|OutsideEnvironment'`

Expected: current signatures accept arbitrary prefix/key.

- [ ] **Step 3: Change service APIs**

Add the testable adapter boundary used by both services:

~~~go
type serviceRegistryStore interface {
    Get(ctx context.Context, key string) (*clientv3.GetResponse, error)
    GetWithPrefix(ctx context.Context, prefix string, limit int64) (*clientv3.GetResponse, error)
    Put(ctx context.Context, key, value string) (*clientv3.PutResponse, error)
    PutWithLease(ctx context.Context, key, value string, leaseID clientv3.LeaseID) (*clientv3.PutResponse, error)
}

var _ serviceRegistryStore = (*etcd.Client)(nil)
~~~

Make both constructors accept `serviceRegistryStore`, then change their APIs:

~~~go
func (s *GatewayService) ListServices(ctx context.Context, env *domain.Environment) ([]ServiceGroup, error)
func (s *GatewayService) UpdateInstanceStatus(ctx context.Context, env *domain.Environment, key, status string) error
func (s *GrpcServiceManager) ListServices(ctx context.Context, env *domain.Environment) ([]GrpcServiceGroup, error)
func (s *GrpcServiceManager) UpdateInstanceStatus(ctx context.Context, env *domain.Environment, key, status string) error
~~~

Each method calls `domain.RequireEnvironmentAccess`. List derives its prefix from the environment. Normalize the derived prefix to exactly one trailing slash, and update checks `strings.HasPrefix(key, derivedPrefix)` before etcd access. The fake store implements all four interface methods and increments counters so the tests prove authorization happens before I/O.

Add `Key string \`json:"key"\`` to both `ServiceInstance` and `GrpcInstance`, and set it from `string(kv.Key)` while listing. Status updates can then use the exact server-returned key instead of reconstructing one from untrusted payload fields.

- [ ] **Step 4: Change handlers**

Inject `EnvironmentService` into gateway and gRPC handlers. Query uses `env`; update JSON uses:

~~~go
var req struct {
    Env string `json:"env" binding:"required"`
    Key string `json:"key" binding:"required"`
    Status string `json:"status" binding:"required,oneof=up down"`
}
~~~

Resolve environment by name and pass it to the service. Map `ErrEnvironmentForbidden` to `CodeForbidden`.

- [ ] **Step 5: Verify GREEN and commit backend**

Run: `gofmt -w internal/handler/gateway.go internal/handler/grpc.go internal/service/gateway_service.go internal/service/grpc_service.go internal/service/service_registry_store.go internal/service/service_registry_access_test.go cmd/server/main.go && go test ./internal/service ./internal/handler`

Expected: PASS.

~~~bash
git add internal/handler/gateway.go internal/handler/grpc.go internal/service/gateway_service.go internal/service/grpc_service.go internal/service/service_registry_store.go internal/service/service_registry_access_test.go cmd/server/main.go
git commit -m "fix: constrain service registry operations by environment"
~~~

### Task 5: Gateway and gRPC Frontend API

**Files:**
- Modify: `web/src/api/gateway.ts`
- Modify: `web/src/api/grpc.ts`
- Modify: `web/src/types/index.ts`
- Modify: `web/src/pages/gateway/index.tsx`
- Modify: `web/src/pages/grpc/index.tsx`

- [ ] **Step 1: Change API clients**

~~~ts
list: (env: string) =>
  request<ServiceGroup[]>(client.get('/gateway', { params: { env } })),
updateStatus: (env: string, key: string, status: 'up' | 'down') =>
  request<null>(client.put('/gateway/status', { env, key, status })),
~~~

Apply the equivalent gRPC signatures.

- [ ] **Step 2: Change pages**

Remove `getPrefix`. Fetch with `currentEnv.name`. Add `key: string` to the two instance types, use the exact `instance.key` returned by the backend for status requests, and pass `currentEnv.name` as `env`. Disable fetching/updating when no current environment is selected.

- [ ] **Step 3: Verify and commit**

Run: `cd web && npx tsc --noEmit --incremental false && npx vite build`

Expected: type check and build pass.

~~~bash
git add web/src/api/gateway.ts web/src/api/grpc.ts web/src/types/index.ts web/src/pages/gateway/index.tsx web/src/pages/grpc/index.tsx
git commit -m "fix: send environment for registry operations"
~~~

### Task 6: Transactional Super-Admin Transfer

**Files:**
- Modify: `internal/service/user_service.go`
- Create: `internal/service/user_service_test.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write a failing rollback test**

Use a fake repository backed by a user map and a fake transaction manager that snapshots and restores the map:

~~~go
func TestTransferSuperRollsBackWhenTargetUpdateFails(t *testing.T) {
    currentID, targetID, roleID := uuid.New(), uuid.New(), uuid.New()
    repo := newTransactionalUserRepo(currentID, targetID)
    repo.failUpdateID = targetID
    svc := NewUserService(repo, &fakeRoleRepo{roleID: roleID}, newSnapshotTxManager(repo))
    err := svc.TransferSuper(context.Background(), currentID, targetID, roleID)
    if err == nil {
        t.Fatal("TransferSuper() error = nil")
    }
    if !repo.users[currentID].IsSuper || repo.users[targetID].IsSuper {
        t.Fatalf("transaction was not rolled back: %+v", repo.users)
    }
}
~~~

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service -run '^TestTransferSuperRollsBack'`

Expected: old super remains demoted because no transaction is used.

- [ ] **Step 3: Inject and use TransactionManager**

~~~go
type UserService struct {
    userRepo domain.UserRepository
    roleRepo domain.RoleRepository
    txManager domain.TransactionManager
}

func NewUserService(userRepo domain.UserRepository, roleRepo domain.RoleRepository, txManager domain.TransactionManager) *UserService
~~~

Validate the downgrade role, then perform both user reads and both updates inside:

~~~go
return s.txManager.WithTransaction(ctx, func(txCtx context.Context) error {
    current, err := s.userRepo.GetByID(txCtx, currentUserID)
    // validate current and target, then update both using txCtx
    return err
})
~~~

- [ ] **Step 4: Verify GREEN and commit**

Run: `gofmt -w internal/service/user_service.go internal/service/user_service_test.go cmd/server/main.go && go test ./internal/service`

Expected: PASS.

~~~bash
git add internal/service/user_service.go internal/service/user_service_test.go cmd/server/main.go
git commit -m "fix: transfer super admin transactionally"
~~~

### Task 7: Authorization Verification

**Files:**
- Verify all authorization files above.

- [ ] **Step 1: Run backend verification**

Run:

~~~bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
~~~

Expected: all commands exit 0.

- [ ] **Step 2: Run frontend and deployment verification**

Run:

~~~bash
cd web && npx tsc --noEmit --incremental false && npx vite build
cd .. && helm lint deploy/helm
~~~

Expected: type check/build pass and Helm reports 0 failed charts.
