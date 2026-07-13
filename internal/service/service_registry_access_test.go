package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

type fakeServiceRegistryStore struct {
	getCalls           int
	getWithPrefixCalls int
	prefixResponse     *clientv3.GetResponse
}

func (s *fakeServiceRegistryStore) Get(context.Context, string) (*clientv3.GetResponse, error) {
	s.getCalls++
	return &clientv3.GetResponse{}, nil
}

func (s *fakeServiceRegistryStore) GetWithPrefix(context.Context, string, int64) (*clientv3.GetResponse, error) {
	s.getWithPrefixCalls++
	if s.prefixResponse != nil {
		return s.prefixResponse, nil
	}
	return &clientv3.GetResponse{}, nil
}

func (s *fakeServiceRegistryStore) Put(context.Context, string, string) (*clientv3.PutResponse, error) {
	return &clientv3.PutResponse{}, nil
}

func (s *fakeServiceRegistryStore) PutWithLease(context.Context, string, string, clientv3.LeaseID) (*clientv3.PutResponse, error) {
	return &clientv3.PutResponse{}, nil
}

func TestGatewayRejectsKeyOutsideEnvironmentPrefix(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), KeyPrefix: "/prod/", GatewayPrefix: "gw-services/"}
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{env.ID}})
	store := &fakeServiceRegistryStore{}
	svc := NewGatewayService(store)

	err := svc.UpdateInstanceStatus(ctx, env, "/other/gw-services/app/1", "down")

	if !errors.Is(err, domain.ErrEnvironmentForbidden) {
		t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
	}
	if store.getCalls != 0 {
		t.Fatal("etcd must not be called for foreign key")
	}
}

func TestGrpcRejectsKeyOutsideEnvironmentPrefix(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), KeyPrefix: "/prod/", GrpcPrefix: "grpc-services/"}
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{env.ID}})
	store := &fakeServiceRegistryStore{}
	svc := NewGrpcServiceManager(store)

	err := svc.UpdateInstanceStatus(ctx, env, "/other/grpc-services/app/1", "down")

	if !errors.Is(err, domain.ErrEnvironmentForbidden) {
		t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
	}
	if store.getCalls != 0 {
		t.Fatal("etcd must not be called for foreign key")
	}
}

func TestGatewayListRejectsUnauthorizedEnvironment(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), KeyPrefix: "/prod/", GatewayPrefix: "gw-services/"}
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{uuid.New()}})
	store := &fakeServiceRegistryStore{}
	svc := NewGatewayService(store)

	_, err := svc.ListServices(ctx, env)

	if !errors.Is(err, domain.ErrEnvironmentForbidden) {
		t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
	}
	if store.getWithPrefixCalls != 0 {
		t.Fatal("etcd must not be called for unauthorized environment")
	}
}

func TestGrpcListRejectsUnauthorizedEnvironment(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), KeyPrefix: "/prod/", GrpcPrefix: "grpc-services/"}
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{uuid.New()}})
	store := &fakeServiceRegistryStore{}
	svc := NewGrpcServiceManager(store)

	_, err := svc.ListServices(ctx, env)

	if !errors.Is(err, domain.ErrEnvironmentForbidden) {
		t.Fatalf("error = %v, want ErrEnvironmentForbidden", err)
	}
	if store.getWithPrefixCalls != 0 {
		t.Fatal("etcd must not be called for unauthorized environment")
	}
}

func TestGatewayListReturnsExactEtcdKey(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), KeyPrefix: "/prod/", GatewayPrefix: "gw-services/"}
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{env.ID}})
	store := &fakeServiceRegistryStore{prefixResponse: &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{{
		Key:   []byte("/prod/gw-services/app/instance-1"),
		Value: []byte(`{"id":"instance-1","service_name":"app","status":"up"}`),
	}}}}

	groups, err := NewGatewayService(store).ListServices(ctx, env)

	if err != nil {
		t.Fatalf("ListServices() error = %v", err)
	}
	if len(groups) != 1 || len(groups[0].Instances) != 1 {
		t.Fatalf("groups = %+v", groups)
	}
	if got := groups[0].Instances[0].Key; got != "/prod/gw-services/app/instance-1" {
		t.Fatalf("key = %q", got)
	}
}

func TestGrpcListReturnsExactEtcdKey(t *testing.T) {
	env := &domain.Environment{ID: uuid.New(), KeyPrefix: "/prod/", GrpcPrefix: "grpc-services/"}
	ctx := domain.WithEnvironmentScope(context.Background(), domain.EnvironmentScope{AllowedIDs: []uuid.UUID{env.ID}})
	store := &fakeServiceRegistryStore{prefixResponse: &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{{
		Key:   []byte("/prod/grpc-services/app/instance-1"),
		Value: []byte(`{"instance_id":"instance-1","service_name":"app","status":"up"}`),
	}}}}

	groups, err := NewGrpcServiceManager(store).ListServices(ctx, env)

	if err != nil {
		t.Fatalf("ListServices() error = %v", err)
	}
	if len(groups) != 1 || len(groups[0].Instances) != 1 {
		t.Fatalf("groups = %+v", groups)
	}
	if got := groups[0].Instances[0].Key; got != "/prod/grpc-services/app/instance-1" {
		t.Fatalf("key = %q", got)
	}
}
