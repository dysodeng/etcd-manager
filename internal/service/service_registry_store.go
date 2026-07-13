package service

import (
	"context"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dysodeng/etcd-manager/internal/etcd"
)

type serviceRegistryStore interface {
	Get(ctx context.Context, key string) (*clientv3.GetResponse, error)
	GetWithPrefix(ctx context.Context, prefix string, limit int64) (*clientv3.GetResponse, error)
	Put(ctx context.Context, key, value string) (*clientv3.PutResponse, error)
	PutWithLease(ctx context.Context, key, value string, leaseID clientv3.LeaseID) (*clientv3.PutResponse, error)
}

func serviceRegistryPrefix(keyPrefix, servicePrefix string) string {
	return strings.TrimRight(keyPrefix, "/") + "/" + strings.Trim(servicePrefix, "/") + "/"
}

var _ serviceRegistryStore = (*etcd.Client)(nil)
