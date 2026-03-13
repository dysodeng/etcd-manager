package service

import (
	"context"

	"github.com/dysodeng/config-center/internal/etcd"
)

type KVService struct {
	etcdClient *etcd.Client
}

func NewKVService(etcdClient *etcd.Client) *KVService {
	return &KVService{etcdClient: etcdClient}
}

type KVItem struct {
	Key            string `json:"key"`
	Value          string `json:"value"`
	CreateRevision int64  `json:"create_revision"`
	ModRevision    int64  `json:"mod_revision"`
	Version        int64  `json:"version"`
}

func (s *KVService) Get(ctx context.Context, key string) (*KVItem, error) {
	resp, err := s.etcdClient.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	kv := resp.Kvs[0]
	return &KVItem{
		Key:            string(kv.Key),
		Value:          string(kv.Value),
		CreateRevision: kv.CreateRevision,
		ModRevision:    kv.ModRevision,
		Version:        kv.Version,
	}, nil
}

func (s *KVService) List(ctx context.Context, prefix string, limit int64) ([]KVItem, error) {
	resp, err := s.etcdClient.GetWithPrefix(ctx, prefix, limit)
	if err != nil {
		return nil, err
	}
	items := make([]KVItem, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		items = append(items, KVItem{
			Key:            string(kv.Key),
			Value:          string(kv.Value),
			CreateRevision: kv.CreateRevision,
			ModRevision:    kv.ModRevision,
			Version:        kv.Version,
		})
	}
	return items, nil
}

func (s *KVService) Put(ctx context.Context, key, value string) error {
	_, err := s.etcdClient.Put(ctx, key, value)
	return err
}

func (s *KVService) Delete(ctx context.Context, key string) error {
	_, err := s.etcdClient.Delete(ctx, key)
	return err
}
