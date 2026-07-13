# Config Consistency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make configuration writes concurrency-safe and keep etcd state recoverable when revision persistence fails.

**Architecture:** Add conditional transaction operations to the etcd adapter and depend on a narrow interface from ConfigService. Config writes use compare-and-swap, persist revision history, and conditionally compensate when persistence fails. Rollback validates ownership and import reuses the same upsert path.

**Tech Stack:** Go 1.25, etcd client v3 transactions, repository interfaces, Gin error mapping, Go fakes and table-driven tests.

---

## File Structure

- Create `internal/etcd/config_store.go`: narrow ConfigStore interface plus conditional create/update/delete and compensation primitives implemented by Client.
- Create `internal/service/config_errors.go`: typed conflict, revision, persistence, and inconsistency errors.
- Modify `internal/service/config_service.go`: CAS writes, revision persistence, compensation, rollback ownership, and import reuse.
- Create `internal/service/config_consistency_test.go`: fake store/repository tests for every state transition.
- Modify `internal/handler/config_center.go`, `internal/response/response.go`, `internal/handler/response.go`: typed error mapping and conflict code.
- Modify `cmd/server/main.go`: constructor wiring after removing the unused ConfigService transaction manager.

### Task 1: Typed Consistency Errors

**Files:**
- Create: `internal/service/config_errors.go`
- Create: `internal/service/config_errors_test.go`

- [ ] **Step 1: Write failing error tests**

~~~go
package service

import (
    "errors"
    "testing"
)

func TestConfigConsistencyErrorsSupportErrorsIs(t *testing.T) {
    err := &ConfigPersistenceError{Operation: "update", Err: errors.New("database unavailable"), Compensated: true}
    if !errors.Is(err, ErrConfigPersistence) {
        t.Fatalf("error = %v, want ErrConfigPersistence", err)
    }
    inconsistent := &ConfigPersistenceError{Operation: "update", Err: errors.New("database unavailable"), CompensationErr: errors.New("compare failed")}
    if !errors.Is(inconsistent, ErrConfigInconsistent) {
        t.Fatalf("error = %v, want ErrConfigInconsistent", inconsistent)
    }
    compareMiss := &ConfigPersistenceError{Operation: "update", Err: errors.New("database unavailable"), Compensated: false}
    if !errors.Is(compareMiss, ErrConfigInconsistent) {
        t.Fatalf("error = %v, want ErrConfigInconsistent", compareMiss)
    }
}
~~~

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service -run '^TestConfigConsistencyErrors'`

Expected: types are undefined.

- [ ] **Step 3: Define typed service errors**

~~~go
var (
    ErrKeyExists = errors.New("key already exists")
    ErrKeyNotFound = errors.New("key not found")
    ErrConfigConflict = errors.New("configuration changed concurrently")
    ErrConfigPersistence = errors.New("configuration revision persistence failed")
    ErrConfigInconsistent = errors.New("configuration state may be inconsistent")
    ErrRevisionNotFound = errors.New("revision not found")
)
~~~

`ConfigPersistenceError` contains `Operation`, the revision repository `Err`, `Compensated`, and `CompensationErr`. Its `Is` returns `ErrConfigInconsistent` whenever compensation did not succeed (including a compare miss with nil compensation error), and `ErrConfigPersistence` only when `Compensated` is true. Its `Error` includes the operation and compensation outcome; `Unwrap` returns the repository error for diagnostics without exposing it in handlers.

- [ ] **Step 4: Verify GREEN and commit**

Run: `gofmt -w internal/service/config_errors.go internal/service/config_errors_test.go && go test ./internal/service -run '^TestConfigConsistencyErrors'`

~~~bash
git add internal/service/config_errors.go internal/service/config_errors_test.go
git commit -m "feat: define config consistency boundaries"
~~~

### Task 2: etcd Conditional Operations

**Files:**
- Create: `internal/etcd/config_store.go`

- [ ] **Step 1: Define the adapter-owned store contract**

Keep the contract in `internal/etcd` so `service` can depend on it without creating an `etcd -> service -> etcd` import cycle. Use `GetConfig` rather than `Get`, because Client already exposes `Get` with a client-v3 response signature:

~~~go
type ConfigSnapshot struct {
    Exists bool
    Value string
    ModRevision int64
    LeaseID int64
}

type ConditionalResult struct {
    Succeeded bool
    Revision int64
}

type ConfigStore interface {
    GetWithPrefix(ctx context.Context, prefix string, limit int64) (*clientv3.GetResponse, error)
    GetConfig(ctx context.Context, key string) (ConfigSnapshot, error)
    CreateIfAbsent(ctx context.Context, key, value string) (ConditionalResult, error)
    PutIfModRevision(ctx context.Context, key, value string, expectedModRevision int64) (ConditionalResult, error)
    DeleteIfModRevision(ctx context.Context, key string, expectedModRevision int64) (ConditionalResult, error)
    DeleteIfModRevisionForCompensation(ctx context.Context, key string, writtenRevision int64) (bool, error)
    RestoreIfModRevision(ctx context.Context, key string, snapshot ConfigSnapshot, writtenRevision int64) (bool, error)
    RestoreIfAbsent(ctx context.Context, key string, snapshot ConfigSnapshot) (bool, error)
}
~~~

- [ ] **Step 2: Add compile-time interface assertion**

At the bottom of `internal/etcd/config_store.go`:

~~~go
var _ ConfigStore = (*Client)(nil)
~~~

Run: `go test ./internal/etcd`

Expected: build fails because Client lacks the conditional methods.

- [ ] **Step 3: Implement snapshot and conditional create**

~~~go
func (c *Client) CreateIfAbsent(ctx context.Context, key, value string) (ConditionalResult, error) {
    resp, err := c.cli.Txn(ctx).
        If(clientv3.Compare(clientv3.Version(key), "=", 0)).
        Then(clientv3.OpPut(key, value)).
        Commit()
    if err != nil {
        return ConditionalResult{}, err
    }
    return ConditionalResult{Succeeded: resp.Succeeded, Revision: resp.Header.Revision}, nil
}
~~~

`GetConfig` maps the first KV into `ConfigSnapshot`, including `ModRevision` and `LeaseID`, and returns `Exists: false` for an empty response.

- [ ] **Step 4: Implement update/delete and compensation**

Update compares `ModRevision(key)` with the expected revision. Delete uses the same comparison.

Create compensation compares `ModRevision(key)` with the write revision before deleting. Update compensation uses the same comparison before restoring the previous snapshot. Delete compensation compares `Version(key) == 0` before restoring the previous snapshot. Both restore methods attach the original lease when `LeaseID != 0`.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w internal/etcd/config_store.go && go test ./internal/etcd ./internal/service`

Expected: interface assertion compiles.

~~~bash
git add internal/etcd/config_store.go
git commit -m "feat: add conditional etcd config operations"
~~~

### Task 3: CAS Create and Update with Compensation

**Files:**
- Modify: `internal/service/config_service.go`
- Create: `internal/service/config_consistency_test.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Write failing create/conflict tests**

Using a `fakeConfigStore` implementing every `etcd.ConfigStore` method (including a `GetWithPrefix` stub for list/export tests) and a fake revision repository:

~~~go
func TestConfigCreateReturnsKeyExistsOnCompareFailure(t *testing.T) {
    store := &fakeConfigStore{createResult: etcd.ConditionalResult{Succeeded: false}}
    svc := newConsistencyTestService(store, successfulRevisionRepo())
    err := svc.Create(authorizedContext(), "dev", "app.yaml", "name: app", "test", uuid.New())
    if !errors.Is(err, ErrKeyExists) {
        t.Fatalf("error = %v, want ErrKeyExists", err)
    }
    if store.putCalls != 0 {
        t.Fatal("fallback put must not occur")
    }
}

func TestConfigUpdateReturnsConflictWhenRevisionChanges(t *testing.T) {
    store := &fakeConfigStore{
        snapshot: etcd.ConfigSnapshot{Exists: true, Value: "old", ModRevision: 10},
        putResult: etcd.ConditionalResult{Succeeded: false},
    }
    svc := newConsistencyTestService(store, successfulRevisionRepo())
    err := svc.Update(authorizedContext(), "dev", "app.yaml", "name: new", "test", uuid.New())
    if !errors.Is(err, ErrConfigConflict) {
        t.Fatalf("error = %v, want ErrConfigConflict", err)
    }
}
~~~

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service -run 'TestConfig(CreateReturns|UpdateReturns)'`

Expected: current service uses Get/Put and cannot produce typed CAS results.

- [ ] **Step 3: Refactor ConfigService constructor**

~~~go
type ConfigService struct {
    configStore etcd.ConfigStore
    envRepo domain.EnvironmentRepository
    revisionRepo domain.ConfigRevisionRepository
}

func NewConfigService(store etcd.ConfigStore, envRepo domain.EnvironmentRepository, revisionRepo domain.ConfigRevisionRepository) *ConfigService
~~~

Remove the unused ConfigService `txManager` and update `main`.

- [ ] **Step 4: Implement create/update state machines**

Create validates and authorizes, calls `CreateIfAbsent`, returns `ErrKeyExists` when comparison fails, then persists its revision.

Update calls `GetConfig`. If the snapshot exists, call `PutIfModRevision`; if absent, call `CreateIfAbsent`. A failed comparison returns `ErrConfigConflict`.

After revision repository failure:

~~~go
compensated, compensationErr := s.compensateWrite(ctx, operation, fullKey, before, result.Revision)
return newConfigPersistenceError(operation, revisionErr, compensated, compensationErr)
~~~

- [ ] **Step 5: Write and pass compensation tests**

Cover:

~~~go
func TestConfigCreateCompensatesWhenRevisionPersistenceFails(t *testing.T)
func TestConfigUpdateRestoresPreviousValueWhenRevisionPersistenceFails(t *testing.T)
func TestConfigUpdateReportsInconsistentWhenCompensationCompareFails(t *testing.T)
~~~

Each test asserts exact fake-store calls, returned sentinel, and no false success.

- [ ] **Step 6: Verify GREEN and commit**

Run: `gofmt -w internal/service/config_service.go internal/service/config_consistency_test.go cmd/server/main.go && go test ./internal/service`

~~~bash
git add internal/service/config_service.go internal/service/config_consistency_test.go cmd/server/main.go
git commit -m "fix: make config writes concurrency safe"
~~~

### Task 4: CAS Delete and Rollback Ownership

**Files:**
- Modify: `internal/service/config_service.go`
- Modify: `internal/service/config_consistency_test.go`

- [ ] **Step 1: Write failing delete and rollback tests**

~~~go
func TestConfigDeleteRestoresValueWhenRevisionPersistenceFails(t *testing.T)
func TestConfigDeleteReturnsConflictWhenRevisionChanges(t *testing.T)
func TestRollbackRejectsRevisionFromAnotherEnvironment(t *testing.T)
func TestRollbackRejectsRevisionFromAnotherKey(t *testing.T)
func TestRollbackRejectsDeleteRevision(t *testing.T)
~~~

The rollback mismatch tests expect `ErrRevisionNotFound` and assert no store write.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service -run 'Delete|RollbackRejects'`

Expected: delete is unconditional and rollback accepts mismatched revisions.

- [ ] **Step 3: Implement CAS delete**

Call `GetConfig`, return `ErrKeyNotFound` when absent, conditionally delete by mod revision, persist the delete revision, and call `RestoreIfAbsent` if persistence fails.

- [ ] **Step 4: Implement rollback ownership**

Resolve and authorize the target environment before revision lookup. Then:

~~~go
if rev.EnvironmentID != env.ID || rev.Key != key || rev.Action == "delete" {
    return ErrRevisionNotFound
}
~~~

Call the internal update helper with the resolved environment.

- [ ] **Step 5: Verify GREEN and commit**

Run: `gofmt -w internal/service/config_service.go internal/service/config_consistency_test.go && go test ./internal/service`

~~~bash
git add internal/service/config_service.go internal/service/config_consistency_test.go
git commit -m "fix: validate delete and rollback state"
~~~

### Task 5: Import Reuses Consistent Upsert

**Files:**
- Modify: `internal/service/config_service.go`
- Modify: `internal/service/config_service_test.go`
- Modify: `internal/service/config_consistency_test.go`

- [ ] **Step 1: Write failing import tests**

~~~go
func TestImportDoesNotCountStoreReadFailureAsSuccess(t *testing.T)
func TestImportDoesNotCountRevisionFailureAsSuccess(t *testing.T)
func TestImportReportsConflictReason(t *testing.T)
func TestImportDryRunOnlyResolvesEnvironment(t *testing.T)
~~~

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service -run '^TestImport'`

Expected: current import ignores read/revision errors and increments Success.

- [ ] **Step 3: Implement import through internal upsert**

Resolve and authorize the environment once. Validate all values. For non-dry-run, call the same internal update/upsert helper used by Update with comment `import`. Append `fmt.Sprintf("%s: %v", key, err)` on failure and increment Success only on nil error.

For dry-run, resolve and authorize the environment, then return after syntax validation. Assert one environment lookup and zero config-store/revision-repository calls. This intentionally reconciles the consistency design with the authorization rule that every import request, including dry-run, must be environment-scoped.

- [ ] **Step 4: Verify GREEN and commit**

Run: `gofmt -w internal/service/config_service.go internal/service/config_service_test.go internal/service/config_consistency_test.go && go test ./internal/service`

~~~bash
git add internal/service/config_service.go internal/service/config_service_test.go internal/service/config_consistency_test.go
git commit -m "fix: preserve consistency during config import"
~~~

### Task 6: Typed Handler Error Mapping

**Files:**
- Modify: `internal/response/response.go`
- Modify: `internal/handler/response.go`
- Modify: `internal/handler/config_center.go`
- Modify: `internal/handler/config_center_test.go`

- [ ] **Step 1: Write failing mapping tests**

~~~go
func TestConfigWriteErrorCode(t *testing.T) {
    tests := []struct {
        err error
        code int
    }{
        {err: service.ErrKeyExists, code: CodeKeyExists},
        {err: service.ErrKeyNotFound, code: CodeKeyNotFound},
        {err: service.ErrConfigConflict, code: CodeConfigConflict},
        {err: domain.ErrEnvironmentForbidden, code: CodeForbidden},
        {err: service.ErrRevisionNotFound, code: CodeRevisionNotFound},
        {err: &service.ConfigPersistenceError{Err: errors.New("db"), Compensated: true}, code: CodeInternalError},
        {err: &service.ConfigPersistenceError{Err: errors.New("db"), CompensationErr: errors.New("etcd")}, code: CodeInternalError},
    }
    // table assertion
}
~~~

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/handler -run '^TestConfigWriteErrorCode$'`

Expected: conflict code and typed mappings do not exist.

- [ ] **Step 3: Add code and errors.Is/As mapping**

Add `CodeConfigConflict = 20006`. Replace every config handler string comparison with one classifier using `errors.Is/As`. Keep parser validation mapped to `CodeParamInvalid`.

- [ ] **Step 4: Verify GREEN and commit**

Run: `gofmt -w internal/response/response.go internal/handler/response.go internal/handler/config_center.go internal/handler/config_center_test.go && go test ./internal/handler`

~~~bash
git add internal/response/response.go internal/handler/response.go internal/handler/config_center.go internal/handler/config_center_test.go
git commit -m "fix: classify config consistency errors"
~~~

### Task 7: Complete Verification

**Files:**
- Verify every file changed by both implementation plans.

- [ ] **Step 1: Verify formatting and diffs**

Run:

~~~bash
test -z "$(gofmt -l internal cmd)"
git diff --check
~~~

Expected: no output.

- [ ] **Step 2: Run backend verification**

Run:

~~~bash
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
~~~

Expected: all commands exit 0.

- [ ] **Step 3: Run frontend and deployment verification**

Run:

~~~bash
cd web && npx tsc --noEmit --incremental false && npx vite build
cd .. && helm lint deploy/helm
~~~

Expected: frontend builds and Helm reports 0 failed charts.

- [ ] **Step 4: Review branch state**

Run: `git status --short --branch && git log --oneline main..HEAD`

Expected: clean feature branch containing only high-priority hardening changes and plan documents.
