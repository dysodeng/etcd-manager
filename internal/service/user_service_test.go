package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type transactionalUserRepository struct {
	users        map[uuid.UUID]*domain.User
	failUpdateID uuid.UUID
}

func newTransactionalUserRepository(users ...*domain.User) *transactionalUserRepository {
	repo := &transactionalUserRepository{users: make(map[uuid.UUID]*domain.User, len(users))}
	for _, user := range users {
		copy := *user
		repo.users[user.ID] = &copy
	}
	return repo
}

func (r *transactionalUserRepository) Create(context.Context, *domain.User) error { panic("not used") }
func (r *transactionalUserRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copy := *user
	return &copy, nil
}
func (r *transactionalUserRepository) GetByUsername(context.Context, string) (*domain.User, error) {
	panic("not used")
}
func (r *transactionalUserRepository) List(context.Context, int, int) ([]domain.User, int64, error) {
	panic("not used")
}
func (r *transactionalUserRepository) Update(_ context.Context, user *domain.User) error {
	if user.ID == r.failUpdateID {
		return errors.New("update failed")
	}
	copy := *user
	r.users[user.ID] = &copy
	return nil
}
func (r *transactionalUserRepository) Delete(context.Context, uuid.UUID) error { panic("not used") }
func (r *transactionalUserRepository) CountByRoleID(context.Context, uuid.UUID) (int64, error) {
	panic("not used")
}
func (r *transactionalUserRepository) GetSuperAdmin(context.Context) (*domain.User, error) {
	panic("not used")
}

type transferRoleRepository struct {
	roleID uuid.UUID
}

func (r *transferRoleRepository) Create(context.Context, *domain.Role) error { panic("not used") }
func (r *transferRoleRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Role, error) {
	if id != r.roleID {
		return nil, errors.New("not found")
	}
	return &domain.Role{ID: id}, nil
}
func (r *transferRoleRepository) GetByName(context.Context, string) (*domain.Role, error) {
	panic("not used")
}
func (r *transferRoleRepository) List(context.Context, int, int) ([]domain.Role, int64, error) {
	panic("not used")
}
func (r *transferRoleRepository) Update(context.Context, *domain.Role) error { panic("not used") }
func (r *transferRoleRepository) Delete(context.Context, uuid.UUID) error    { panic("not used") }
func (r *transferRoleRepository) GetPermissions(context.Context, uuid.UUID) ([]domain.RolePermission, error) {
	panic("not used")
}
func (r *transferRoleRepository) SetPermissions(context.Context, uuid.UUID, []domain.RolePermission) error {
	panic("not used")
}
func (r *transferRoleRepository) GetEnvironmentIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	panic("not used")
}
func (r *transferRoleRepository) SetEnvironments(context.Context, uuid.UUID, []uuid.UUID) error {
	panic("not used")
}
func (r *transferRoleRepository) DeleteEnvironmentByEnvID(context.Context, uuid.UUID) error {
	panic("not used")
}

type snapshotTransactionManager struct {
	repo *transactionalUserRepository
}

func (m *snapshotTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	snapshot := make(map[uuid.UUID]*domain.User, len(m.repo.users))
	for id, user := range m.repo.users {
		copy := *user
		snapshot[id] = &copy
	}
	if err := fn(ctx); err != nil {
		m.repo.users = snapshot
		return err
	}
	return nil
}

func TestTransferSuperRollsBackWhenTargetUpdateFails(t *testing.T) {
	currentID, targetID, roleID := uuid.New(), uuid.New(), uuid.New()
	repo := newTransactionalUserRepository(
		&domain.User{ID: currentID, IsSuper: true},
		&domain.User{ID: targetID, IsSuper: false},
	)
	repo.failUpdateID = targetID
	svc := NewUserService(repo, &transferRoleRepository{roleID: roleID}, &snapshotTransactionManager{repo: repo})

	err := svc.TransferSuper(context.Background(), currentID, targetID, roleID)

	if err == nil {
		t.Fatal("TransferSuper() error = nil")
	}
	if !repo.users[currentID].IsSuper || repo.users[targetID].IsSuper {
		t.Fatalf("transaction was not rolled back: %+v", repo.users)
	}
}
