package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/etcd"
)

type fakeConfigStore struct {
	prefixResponse *clientv3.GetResponse
	snapshot       etcd.ConfigSnapshot
	getErr         error
	createResult   etcd.ConditionalResult
	createErr      error
	putResult      etcd.ConditionalResult
	putErr         error
	deleteResult   etcd.ConditionalResult
	deleteErr      error
	compensated    bool
	compensateErr  error

	getCalls                int
	createCalls             int
	putCalls                int
	deleteCalls             int
	deleteCompensationCalls int
	restoreRevisionCalls    int
	restoreAbsentCalls      int
	restoredSnapshot        etcd.ConfigSnapshot
}

func (s *fakeConfigStore) GetWithPrefix(context.Context, string, int64) (*clientv3.GetResponse, error) {
	if s.prefixResponse != nil {
		return s.prefixResponse, nil
	}
	return &clientv3.GetResponse{}, nil
}
func (s *fakeConfigStore) GetConfig(context.Context, string) (etcd.ConfigSnapshot, error) {
	s.getCalls++
	return s.snapshot, s.getErr
}
func (s *fakeConfigStore) CreateIfAbsent(context.Context, string, string) (etcd.ConditionalResult, error) {
	s.createCalls++
	return s.createResult, s.createErr
}
func (s *fakeConfigStore) PutIfModRevision(context.Context, string, string, int64) (etcd.ConditionalResult, error) {
	s.putCalls++
	return s.putResult, s.putErr
}
func (s *fakeConfigStore) DeleteIfModRevision(context.Context, string, int64) (etcd.ConditionalResult, error) {
	s.deleteCalls++
	return s.deleteResult, s.deleteErr
}
func (s *fakeConfigStore) DeleteIfModRevisionForCompensation(context.Context, string, int64) (bool, error) {
	s.deleteCompensationCalls++
	return s.compensated, s.compensateErr
}
func (s *fakeConfigStore) RestoreIfModRevision(_ context.Context, _ string, snapshot etcd.ConfigSnapshot, _ int64) (bool, error) {
	s.restoreRevisionCalls++
	s.restoredSnapshot = snapshot
	return s.compensated, s.compensateErr
}
func (s *fakeConfigStore) RestoreIfAbsent(_ context.Context, _ string, snapshot etcd.ConfigSnapshot) (bool, error) {
	s.restoreAbsentCalls++
	s.restoredSnapshot = snapshot
	return s.compensated, s.compensateErr
}

type fakeConfigRevisionRepository struct {
	createErr   error
	createCalls int
	created     []*domain.ConfigRevision
	revision    *domain.ConfigRevision
}

func (r *fakeConfigRevisionRepository) Create(_ context.Context, revision *domain.ConfigRevision) error {
	r.createCalls++
	copy := *revision
	r.created = append(r.created, &copy)
	return r.createErr
}
func (r *fakeConfigRevisionRepository) ListByKey(context.Context, uuid.UUID, string, int, int) ([]domain.ConfigRevision, int64, error) {
	return nil, 0, nil
}
func (r *fakeConfigRevisionRepository) GetByID(context.Context, uuid.UUID) (*domain.ConfigRevision, error) {
	if r.revision == nil {
		return nil, errors.New("not found")
	}
	copy := *r.revision
	return &copy, nil
}
func (r *fakeConfigRevisionRepository) ListLatestByEnvironment(context.Context, uuid.UUID) ([]domain.ConfigRevision, error) {
	return nil, nil
}

func TestConfigCreateReturnsKeyExistsOnCompareFailure(t *testing.T) {
	store := &fakeConfigStore{createResult: etcd.ConditionalResult{Succeeded: false}}
	svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

	err := svc.Create(authorizedConfigContext(), "dev", "app.yaml", "name: app", "test", uuid.New())

	if !errors.Is(err, ErrKeyExists) {
		t.Fatalf("error = %v, want ErrKeyExists", err)
	}
	if store.putCalls != 0 {
		t.Fatal("fallback put must not occur")
	}
}

func TestConfigListRejectsTooManyItems(t *testing.T) {
	kvs := make([]*mvccpb.KeyValue, MaxConfigListItems+1)
	for i := range kvs {
		kvs[i] = &mvccpb.KeyValue{
			Key:   []byte(fmt.Sprintf("/dev/config/key-%03d", i)),
			Value: []byte("value"),
		}
	}
	store := &fakeConfigStore{prefixResponse: &clientv3.GetResponse{Kvs: kvs}}
	svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

	_, err := svc.List(authorizedConfigContext(), "dev", "")

	if !errors.Is(err, ErrConfigListLimitExceeded) {
		t.Fatalf("error = %v, want ErrConfigListLimitExceeded", err)
	}
}

func TestConfigListRejectsTooManyBytes(t *testing.T) {
	store := &fakeConfigStore{prefixResponse: &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{
		{
			Key:   []byte("/dev/config/large.yaml"),
			Value: bytes.Repeat([]byte("x"), MaxConfigListBytes),
		},
	}}}
	svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

	_, err := svc.List(authorizedConfigContext(), "dev", "")

	if !errors.Is(err, ErrConfigListLimitExceeded) {
		t.Fatalf("error = %v, want ErrConfigListLimitExceeded", err)
	}
}

func TestConfigUpdateReturnsConflictWhenRevisionChanges(t *testing.T) {
	store := &fakeConfigStore{
		snapshot:  etcd.ConfigSnapshot{Exists: true, Value: "old", ModRevision: 10},
		putResult: etcd.ConditionalResult{Succeeded: false},
	}
	svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

	err := svc.Update(authorizedConfigContext(), "dev", "app.yaml", "name: new", "test", uuid.New())

	if !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("error = %v, want ErrConfigConflict", err)
	}
}

func TestConfigCreateCompensatesWhenRevisionPersistenceFails(t *testing.T) {
	store := &fakeConfigStore{
		createResult: etcd.ConditionalResult{Succeeded: true, Revision: 11},
		compensated:  true,
	}
	repo := &fakeConfigRevisionRepository{createErr: errors.New("database unavailable")}
	svc := newConsistencyTestService(store, repo)

	err := svc.Create(authorizedConfigContext(), "dev", "app.yaml", "name: app", "test", uuid.New())

	if !errors.Is(err, ErrConfigPersistence) {
		t.Fatalf("error = %v, want ErrConfigPersistence", err)
	}
	if store.deleteCompensationCalls != 1 {
		t.Fatalf("delete compensation calls = %d", store.deleteCompensationCalls)
	}
}

func TestConfigUpdateRestoresPreviousValueWhenRevisionPersistenceFails(t *testing.T) {
	before := etcd.ConfigSnapshot{Exists: true, Value: "old", ModRevision: 10, LeaseID: 99}
	store := &fakeConfigStore{
		snapshot:    before,
		putResult:   etcd.ConditionalResult{Succeeded: true, Revision: 11},
		compensated: true,
	}
	repo := &fakeConfigRevisionRepository{createErr: errors.New("database unavailable")}
	svc := newConsistencyTestService(store, repo)

	err := svc.Update(authorizedConfigContext(), "dev", "app.yaml", "name: new", "test", uuid.New())

	if !errors.Is(err, ErrConfigPersistence) {
		t.Fatalf("error = %v, want ErrConfigPersistence", err)
	}
	if store.restoreRevisionCalls != 1 || store.restoredSnapshot != before {
		t.Fatalf("restore calls = %d snapshot = %+v", store.restoreRevisionCalls, store.restoredSnapshot)
	}
}

func TestConfigUpdateReportsInconsistentWhenCompensationCompareFails(t *testing.T) {
	store := &fakeConfigStore{
		snapshot:    etcd.ConfigSnapshot{Exists: true, Value: "old", ModRevision: 10},
		putResult:   etcd.ConditionalResult{Succeeded: true, Revision: 11},
		compensated: false,
	}
	repo := &fakeConfigRevisionRepository{createErr: errors.New("database unavailable")}
	svc := newConsistencyTestService(store, repo)

	err := svc.Update(authorizedConfigContext(), "dev", "app.yaml", "name: new", "test", uuid.New())

	if !errors.Is(err, ErrConfigInconsistent) {
		t.Fatalf("error = %v, want ErrConfigInconsistent", err)
	}
}

func TestConfigDeleteRestoresValueWhenRevisionPersistenceFails(t *testing.T) {
	before := etcd.ConfigSnapshot{Exists: true, Value: "old", ModRevision: 10, LeaseID: 99}
	store := &fakeConfigStore{
		snapshot:     before,
		deleteResult: etcd.ConditionalResult{Succeeded: true, Revision: 11},
		compensated:  true,
	}
	repo := &fakeConfigRevisionRepository{createErr: errors.New("database unavailable")}
	svc := newConsistencyTestService(store, repo)

	err := svc.Delete(authorizedConfigContext(), "dev", "app.yaml", uuid.New())

	if !errors.Is(err, ErrConfigPersistence) {
		t.Fatalf("error = %v, want ErrConfigPersistence", err)
	}
	if store.restoreAbsentCalls != 1 || store.restoredSnapshot != before {
		t.Fatalf("restore calls = %d snapshot = %+v", store.restoreAbsentCalls, store.restoredSnapshot)
	}
}

func TestConfigDeleteReturnsConflictWhenRevisionChanges(t *testing.T) {
	store := &fakeConfigStore{
		snapshot:     etcd.ConfigSnapshot{Exists: true, Value: "old", ModRevision: 10},
		deleteResult: etcd.ConditionalResult{Succeeded: false},
	}
	svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

	err := svc.Delete(authorizedConfigContext(), "dev", "app.yaml", uuid.New())

	if !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("error = %v, want ErrConfigConflict", err)
	}
}

func TestRollbackRejectsRevisionOutsideTargetResource(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), Name: "dev", KeyPrefix: "/dev/", ConfigPrefix: "config/"}
	tests := []struct {
		name     string
		revision *domain.ConfigRevision
	}{
		{name: "another environment", revision: &domain.ConfigRevision{EnvironmentID: uuid.New(), Key: "app.yaml", Action: "update", Value: "name: old"}},
		{name: "another key", revision: &domain.ConfigRevision{EnvironmentID: env.ID, Key: "other.yaml", Action: "update", Value: "name: old"}},
		{name: "delete revision", revision: &domain.ConfigRevision{EnvironmentID: env.ID, Key: "app.yaml", Action: "delete"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeConfigStore{}
			repo := &fakeConfigRevisionRepository{revision: tt.revision}
			svc := NewConfigService(store, &configAuthorizationEnvironmentRepository{environment: env}, repo)

			err := svc.Rollback(authorizedConfigContext(), "dev", "app.yaml", uuid.New(), uuid.New())

			if !errors.Is(err, ErrRevisionNotFound) {
				t.Fatalf("error = %v, want ErrRevisionNotFound", err)
			}
			if store.getCalls != 0 || store.putCalls != 0 || store.createCalls != 0 {
				t.Fatalf("store calls: get=%d put=%d create=%d", store.getCalls, store.putCalls, store.createCalls)
			}
		})
	}
}

func TestImportDoesNotCountStoreReadFailureAsSuccess(t *testing.T) {
	store := &fakeConfigStore{getErr: errors.New("etcd unavailable")}
	svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

	result, err := svc.Import(authorizedConfigContext(), "dev", []byte(`{"app.yaml":"name: app"}`), false, uuid.New())

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Success != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0], "etcd unavailable") {
		t.Fatalf("result = %+v", result)
	}
}

func TestImportDoesNotCountRevisionFailureAsSuccess(t *testing.T) {
	store := &fakeConfigStore{
		createResult: etcd.ConditionalResult{Succeeded: true, Revision: 11},
		compensated:  true,
	}
	repo := &fakeConfigRevisionRepository{createErr: errors.New("database unavailable")}
	svc := newConsistencyTestService(store, repo)

	result, err := svc.Import(authorizedConfigContext(), "dev", []byte(`{"app.yaml":"name: app"}`), false, uuid.New())

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Success != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0], "revision persistence failed") {
		t.Fatalf("result = %+v", result)
	}
}

func TestImportReportsConflictReason(t *testing.T) {
	store := &fakeConfigStore{createResult: etcd.ConditionalResult{Succeeded: false}}
	svc := newConsistencyTestService(store, &fakeConfigRevisionRepository{})

	result, err := svc.Import(authorizedConfigContext(), "dev", []byte(`{"app.yaml":"name: app"}`), false, uuid.New())

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Success != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0], ErrConfigConflict.Error()) {
		t.Fatalf("result = %+v", result)
	}
}

func TestImportDryRunDoesNotAccessConfigStoreOrRevisionRepository(t *testing.T) {
	store := &fakeConfigStore{}
	repo := &fakeConfigRevisionRepository{}
	svc := newConsistencyTestService(store, repo)

	result, err := svc.Import(authorizedConfigContext(), "dev", []byte(`{"app.yaml":"name: app"}`), true, uuid.New())

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Success != 1 || store.getCalls != 0 || store.createCalls != 0 || repo.createCalls != 0 {
		t.Fatalf("result=%+v store=%+v revision calls=%d", result, store, repo.createCalls)
	}
}

func newConsistencyTestService(store etcd.ConfigStore, revisionRepo domain.ConfigRevisionRepository) *ConfigService {
	env := &domain.Environment{ID: uuid.New(), Name: "dev", KeyPrefix: "/dev/", ConfigPrefix: "config/"}
	return NewConfigService(store, &configAuthorizationEnvironmentRepository{environment: env}, revisionRepo)
}

func authorizedConfigContext() context.Context {
	return domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{Unrestricted: true})
}
