package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dysodeng/config-center/internal/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type GatewayService struct {
	etcdClient *etcd.Client
}

func NewGatewayService(etcdClient *etcd.Client) *GatewayService {
	return &GatewayService{etcdClient: etcdClient}
}

// ServiceInstance 单个服务实例
type ServiceInstance struct {
	ID           string            `json:"id"`
	ServiceName  string            `json:"service_name"`
	Host         string            `json:"host"`
	Port         int               `json:"port"`
	Weight       int               `json:"weight"`
	Version      string            `json:"version"`
	Status       string            `json:"status"`
	RegisteredAt string            `json:"registered_at"`
	Metadata     map[string]string `json:"metadata"`
}

// ServiceGroup 按服务名分组
type ServiceGroup struct {
	ServiceName    string            `json:"service_name"`
	InstanceCount  int               `json:"instance_count"`
	HealthyCount   int               `json:"healthy_count"`
	UnhealthyCount int              `json:"unhealthy_count"`
	Instances      []ServiceInstance `json:"instances"`
}

// ListServices 列出指定前缀下所有服务，按服务名分组
func (s *GatewayService) ListServices(ctx context.Context, prefix string) ([]ServiceGroup, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.etcdClient.GetWithPrefix(ctx, prefix, 0)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string]*ServiceGroup)

	for _, kv := range resp.Kvs {
		var inst ServiceInstance
		if err := json.Unmarshal(kv.Value, &inst); err != nil {
			continue
		}
		// 从 key 中提取 service_name（倒数第二段）
		if inst.ServiceName == "" {
			parts := strings.Split(string(kv.Key), "/")
			if len(parts) >= 2 {
				inst.ServiceName = parts[len(parts)-2]
			}
		}

		group, ok := groupMap[inst.ServiceName]
		if !ok {
			group = &ServiceGroup{ServiceName: inst.ServiceName}
			groupMap[inst.ServiceName] = group
		}
		group.Instances = append(group.Instances, inst)
		group.InstanceCount++
		if inst.Status == "up" {
			group.HealthyCount++
		} else {
			group.UnhealthyCount++
		}
	}

	groups := make([]ServiceGroup, 0, len(groupMap))
	for _, g := range groupMap {
		sort.Slice(g.Instances, func(i, j int) bool {
			return g.Instances[i].RegisteredAt > g.Instances[j].RegisteredAt
		})
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ServiceName < groups[j].ServiceName
	})
	return groups, nil
}

// UpdateInstanceStatus 更新实例状态，保留原 key 的 lease
func (s *GatewayService) UpdateInstanceStatus(ctx context.Context, key string, status string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.etcdClient.Get(ctx, key)
	if err != nil {
		return err
	}
	if len(resp.Kvs) == 0 {
		return fmt.Errorf("instance not found: %s", key)
	}

	kv := resp.Kvs[0]

	var inst map[string]any
	if err := json.Unmarshal(kv.Value, &inst); err != nil {
		return fmt.Errorf("invalid instance data: %w", err)
	}
	inst["status"] = status

	data, err := json.Marshal(inst)
	if err != nil {
		return err
	}

	leaseID := clientv3.LeaseID(kv.Lease)
	if leaseID != 0 {
		_, err = s.etcdClient.PutWithLease(ctx, key, string(data), leaseID)
	} else {
		_, err = s.etcdClient.Put(ctx, key, string(data))
	}
	return err
}