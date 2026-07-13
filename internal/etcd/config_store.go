package etcd

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type ConfigSnapshot struct {
	Exists      bool
	Value       string
	ModRevision int64
	LeaseID     int64
}

type ConditionalResult struct {
	Succeeded bool
	Revision  int64
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

func (c *Client) GetConfig(ctx context.Context, key string) (ConfigSnapshot, error) {
	resp, err := c.cli.Get(ctx, key)
	if err != nil {
		return ConfigSnapshot{}, err
	}
	if len(resp.Kvs) == 0 {
		return ConfigSnapshot{}, nil
	}
	kv := resp.Kvs[0]
	return ConfigSnapshot{
		Exists:      true,
		Value:       string(kv.Value),
		ModRevision: kv.ModRevision,
		LeaseID:     kv.Lease,
	}, nil
}

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

func (c *Client) PutIfModRevision(ctx context.Context, key, value string, expectedModRevision int64) (ConditionalResult, error) {
	resp, err := c.cli.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", expectedModRevision)).
		Then(clientv3.OpPut(key, value)).
		Commit()
	if err != nil {
		return ConditionalResult{}, err
	}
	return ConditionalResult{Succeeded: resp.Succeeded, Revision: resp.Header.Revision}, nil
}

func (c *Client) DeleteIfModRevision(ctx context.Context, key string, expectedModRevision int64) (ConditionalResult, error) {
	resp, err := c.cli.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", expectedModRevision)).
		Then(clientv3.OpDelete(key)).
		Commit()
	if err != nil {
		return ConditionalResult{}, err
	}
	return ConditionalResult{Succeeded: resp.Succeeded, Revision: resp.Header.Revision}, nil
}

func (c *Client) DeleteIfModRevisionForCompensation(ctx context.Context, key string, writtenRevision int64) (bool, error) {
	resp, err := c.cli.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", writtenRevision)).
		Then(clientv3.OpDelete(key)).
		Commit()
	if err != nil {
		return false, err
	}
	return resp.Succeeded, nil
}

func (c *Client) RestoreIfModRevision(ctx context.Context, key string, snapshot ConfigSnapshot, writtenRevision int64) (bool, error) {
	resp, err := c.cli.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", writtenRevision)).
		Then(configSnapshotPut(key, snapshot)).
		Commit()
	if err != nil {
		return false, err
	}
	return resp.Succeeded, nil
}

func (c *Client) RestoreIfAbsent(ctx context.Context, key string, snapshot ConfigSnapshot) (bool, error) {
	resp, err := c.cli.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(key), "=", 0)).
		Then(configSnapshotPut(key, snapshot)).
		Commit()
	if err != nil {
		return false, err
	}
	return resp.Succeeded, nil
}

func configSnapshotPut(key string, snapshot ConfigSnapshot) clientv3.Op {
	if snapshot.LeaseID != 0 {
		return clientv3.OpPut(key, snapshot.Value, clientv3.WithLease(clientv3.LeaseID(snapshot.LeaseID)))
	}
	return clientv3.OpPut(key, snapshot.Value)
}

var _ ConfigStore = (*Client)(nil)
