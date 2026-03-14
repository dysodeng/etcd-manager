# gRPC 服务管理 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 gRPC 服务的管理功能，按环境变量区分，与现有网关服务管理模式一致。

**Architecture:** 完全复用现有 Gateway 模式：后端新增 GrpcService + GrpcHandler，前端新增 GrpcPage 页面。gRPC 服务实例存储在 etcd 中，key 格式为 `{key_prefix}/{grpc_prefix}/{service_name}/{instance_id}`，value 为 JSON。与 Gateway 的区别在于数据结构不同：gRPC 使用 `address`（单字段）、`tags`（数组）、`register_time`（unix 时间戳）、`instance_id`、`properties` 等字段。

**Tech Stack:** Go (Gin), etcd client v3, React 18, TypeScript, Ant Design

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `internal/service/grpc_service.go` | gRPC 服务业务逻辑（列表、状态更新） |
| Create | `internal/handler/grpc.go` | gRPC 服务 HTTP handler |
| Modify | `internal/handler/router.go:17-18,86-92` | 注册 Grpc handler 和路由 |
| Modify | `cmd/server/main.go:87-98` | 初始化 GrpcService 和 GrpcHandler |
| Create | `web/src/api/grpc.ts` | gRPC 服务 API 客户端 |
| Modify | `web/src/types/index.ts:187-206` | 新增 gRPC 类型定义 |
| Create | `web/src/pages/grpc/index.tsx` | gRPC 服务管理页面 |
| Modify | `web/src/App.tsx:40-41` | 新增 gRPC 路由 |
| Modify | `web/src/layouts/MainLayout.tsx:33-34` | 新增 gRPC 侧边栏菜单项 |

---

## Chunk 1: Backend — Service Layer + Handler + Routing

### Task 1: 创建 gRPC Service 层

**Files:**
- Create: `internal/service/grpc_service.go`

- [ ] **Step 1: 创建 grpc_service.go**

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dysodeng/etcd-manager/internal/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type GrpcServiceManager struct {
	etcdClient *etcd.Client
}

func NewGrpcServiceManager(etcdClient *etcd.Client) *GrpcServiceManager {
	return &GrpcServiceManager{etcdClient: etcdClient}
}

// GrpcInstance 单个 gRPC 服务实例
type GrpcInstance struct {
	ServiceName  string            `json:"service_name"`
	Version      string            `json:"version"`
	Address      string            `json:"address"`
	Env          string            `json:"env"`
	Weight       int               `json:"weight"`
	Tags         []string          `json:"tags"`
	Status       string            `json:"status"`
	RegisterTime int64             `json:"register_time"`
	InstanceID   string            `json:"instance_id"`
	Properties   map[string]string `json:"properties"`
}

// GrpcServiceGroup 按服务名分组
type GrpcServiceGroup struct {
	ServiceName    string         `json:"service_name"`
	InstanceCount  int            `json:"instance_count"`
	HealthyCount   int            `json:"healthy_count"`
	UnhealthyCount int            `json:"unhealthy_count"`
	Instances      []GrpcInstance `json:"instances"`
}

// ListServices 列出指定前缀下所有 gRPC 服务，按服务名分组
func (s *GrpcServiceManager) ListServices(ctx context.Context, prefix string) ([]GrpcServiceGroup, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.etcdClient.GetWithPrefix(ctx, prefix, 0)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string]*GrpcServiceGroup)

	for _, kv := range resp.Kvs {
		var inst GrpcInstance
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
			group = &GrpcServiceGroup{ServiceName: inst.ServiceName}
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

	groups := make([]GrpcServiceGroup, 0, len(groupMap))
	for _, g := range groupMap {
		sort.Slice(g.Instances, func(i, j int) bool {
			return g.Instances[i].RegisterTime > g.Instances[j].RegisterTime
		})
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ServiceName < groups[j].ServiceName
	})
	return groups, nil
}

// UpdateInstanceStatus 更新实例状态，保留原 key 的 lease
func (s *GrpcServiceManager) UpdateInstanceStatus(ctx context.Context, key string, status string) error {
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
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/dysodeng/project/go/etcd-manager && go build ./internal/service/`
Expected: 编译成功，无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/grpc_service.go
git commit -m "feat: add gRPC service manager layer"
```

---

### Task 2: 创建 gRPC Handler

**Files:**
- Create: `internal/handler/grpc.go`

- [ ] **Step 1: 创建 grpc.go**

```go
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type GrpcHandler struct {
	grpcSvc  *service.GrpcServiceManager
	auditSvc *service.AuditService
}

func NewGrpcHandler(grpcSvc *service.GrpcServiceManager, auditSvc *service.AuditService) *GrpcHandler {
	return &GrpcHandler{grpcSvc: grpcSvc, auditSvc: auditSvc}
}

// List 列出所有 gRPC 服务（按服务名分组）
func (h *GrpcHandler) List(c *gin.Context) {
	prefix := c.Query("prefix")
	if prefix == "" {
		Fail(c, CodeParamInvalid, "prefix is required")
		return
	}

	groups, err := h.grpcSvc.ListServices(c.Request.Context(), prefix)
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, groups)
}

// UpdateStatus 更新实例状态（上线/下线）
func (h *GrpcHandler) UpdateStatus(c *gin.Context) {
	var req struct {
		Key    string `json:"key" binding:"required"`
		Status string `json:"status" binding:"required,oneof=up down"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "key and status(up/down) are required")
		return
	}

	if err := h.grpcSvc.UpdateInstanceStatus(c.Request.Context(), req.Key, req.Status); err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}

	action := "deregister"
	if req.Status == "up" {
		action = "register"
	}
	userID, _ := getUserID(c)
	h.auditSvc.Log(c.Request.Context(), userID, action, "grpc_service_instance", req.Key, req.Status, c.ClientIP())

	OK(c, nil)
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/dysodeng/project/go/etcd-manager && go build ./internal/handler/`
Expected: 编译成功，无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/grpc.go
git commit -m "feat: add gRPC service HTTP handler"
```

---

### Task 3: 注册路由 + 初始化依赖

**Files:**
- Modify: `internal/handler/router.go:9-18,86-92`
- Modify: `cmd/server/main.go:87-98`

- [ ] **Step 1: 在 Handlers 结构体中添加 Grpc 字段**

在 `internal/handler/router.go` 的 `Handlers` 结构体中添加：

```go
type Handlers struct {
	Auth         *AuthHandler
	KV           *KVHandler
	ConfigCenter *ConfigCenterHandler
	Watch        *WatchHandler
	Cluster      *ClusterHandler
	User         *UserHandler
	Audit        *AuditHandler
	Gateway      *GatewayHandler
	Grpc         *GrpcHandler
}
```

- [ ] **Step 2: 在 RegisterRoutes 中添加 gRPC 路由**

在 `internal/handler/router.go` 的 `RegisterRoutes` 函数中，gateway 路由块之后添加：

```go
		// gRPC 服务管理（viewer 只读，admin 可下线）
		grpc := auth.Group("/grpc", middleware.RequireAdminForWrite())
		{
			grpc.GET("", h.Grpc.List)
			grpc.PUT("/status", h.Grpc.UpdateStatus)
		}
```

- [ ] **Step 3: 在 main.go 中初始化 GrpcServiceManager 和 GrpcHandler**

在 `cmd/server/main.go` 中，`gatewaySvc` 之后添加：

```go
	grpcSvc := service.NewGrpcServiceManager(etcdClient)
```

在 `handlers` 结构体初始化中添加：

```go
		Grpc:         handler.NewGrpcHandler(grpcSvc, auditSvc),
```

- [ ] **Step 4: 验证编译**

Run: `cd /Users/dysodeng/project/go/etcd-manager && go build ./cmd/server/`
Expected: 编译成功，无错误

- [ ] **Step 5: Commit**

```bash
git add internal/handler/router.go cmd/server/main.go
git commit -m "feat: register gRPC service routes and wire dependencies"
```

---

## Chunk 2: Frontend — Types, API, Page, Routing

### Task 4: 新增 gRPC TypeScript 类型定义 + 共享工具函数

**Files:**
- Modify: `web/src/types/index.ts:187-206`
- Modify: `web/src/utils/index.ts`

- [ ] **Step 1: 在 types/index.ts 末尾添加 gRPC 类型**

在文件末尾（`ServiceGroup` 接口之后）添加：

```typescript
// gRPC Service
export interface GrpcInstance {
  service_name: string
  version: string
  address: string
  env: string
  weight: number
  tags: string[]
  status: 'up' | 'down'
  register_time: number
  instance_id: string
  properties: Record<string, string>
}

export interface GrpcServiceGroup {
  service_name: string
  instance_count: number
  healthy_count: number
  unhealthy_count: number
  instances: GrpcInstance[]
}
```

- [ ] **Step 2: 在 utils/index.ts 中添加 unix 时间戳格式化函数**

在 `web/src/utils/index.ts` 中，`formatTime` 函数之后添加：

```typescript
export function formatUnixTime(ts: number): string {
  if (!ts) return '-'
  return dayjs.unix(ts).format('YYYY-MM-DD HH:mm:ss')
}
```

- [ ] **Step 3: Commit**

```bash
cd web && git add src/types/index.ts src/utils/index.ts
git commit -m "feat: add gRPC service types and formatUnixTime util"
```

---

### Task 5: 创建 gRPC API 客户端

**Files:**
- Create: `web/src/api/grpc.ts`

- [ ] **Step 1: 创建 grpc.ts**

```typescript
import client, { request } from './client'
import type { GrpcServiceGroup } from '@/types'

export const grpcApi = {
  list: (prefix: string) =>
    request<GrpcServiceGroup[]>(client.get('/grpc', { params: { prefix } })),
  updateStatus: (key: string, status: 'up' | 'down') =>
    request<null>(client.put('/grpc/status', { key, status })),
}
```

- [ ] **Step 2: Commit**

```bash
cd web && git add src/api/grpc.ts
git commit -m "feat: add gRPC service API client"
```

---

### Task 6: 创建 gRPC 服务管理页面

**Files:**
- Create: `web/src/pages/grpc/index.tsx`

- [ ] **Step 1: 创建 gRPC 页面组件**

```tsx
import { useEffect, useState } from 'react'
import {
  Card, Table, Button, Space, Tag, Modal, Popconfirm,
  Statistic, Row, Col, Collapse, Badge, Tooltip, Empty, Spin, message,
} from 'antd'
import {
  ReloadOutlined, EyeOutlined,
  StopOutlined, CheckCircleOutlined, CloseCircleOutlined, PlayCircleOutlined,
} from '@ant-design/icons'
import type { GrpcServiceGroup, GrpcInstance } from '@/types'
import { grpcApi } from '@/api/grpc'
import { useAuthStore } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import MonacoEditor from '@/components/MonacoEditor'
import { formatUnixTime } from '@/utils'

export default function GrpcPage() {
  const currentEnv = useEnvironmentStore((s) => s.current)
  const isAdmin = useAuthStore((s) => s.user?.role === 'admin')

  const [groups, setGroups] = useState<GrpcServiceGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [previewJson, setPreviewJson] = useState<string | null>(null)

  const getPrefix = () => {
    if (!currentEnv?.key_prefix) return ''
    const base = currentEnv.key_prefix.endsWith('/')
      ? currentEnv.key_prefix
      : currentEnv.key_prefix + '/'
    return base + (currentEnv.grpc_prefix || 'grpc-services/')
  }

  const fetchData = async () => {
    const prefix = getPrefix()
    if (!prefix) return
    setLoading(true)
    try {
      const data = await grpcApi.list(prefix)
      setGroups(data ?? [])
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (currentEnv?.key_prefix) fetchData()
  }, [currentEnv])

  const handleUpdateStatus = async (instance: GrpcInstance, status: 'up' | 'down') => {
    const key = getPrefix() + instance.service_name + '/' + instance.instance_id
    try {
      await grpcApi.updateStatus(key, status)
      message.success(status === 'up' ? '实例已上线' : '实例已下线')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const totalInstances = groups.reduce((sum, g) => sum + g.instance_count, 0)
  const totalHealthy = groups.reduce((sum, g) => sum + g.healthy_count, 0)

  const instanceColumns = [
    {
      title: '实例ID', dataIndex: 'instance_id', key: 'instance_id',
      render: (id: string) => (
        <span style={{ fontFamily: 'monospace' }}>{id}</span>
      ),
    },
    {
      title: '地址', dataIndex: 'address', key: 'address',
      render: (addr: string) => (
        <span style={{ fontFamily: 'monospace' }}>{addr}</span>
      ),
    },
    { title: '版本', dataIndex: 'version', key: 'version', width: 100 },
    { title: '权重', dataIndex: 'weight', key: 'weight', width: 80 },
    {
      title: '标签', dataIndex: 'tags', key: 'tags',
      render: (tags: string[]) => (
        <Space size={[0, 4]} wrap>
          {(tags || []).map((tag) => (
            <Tag key={tag} color="blue">{tag}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s: string) => {
        if (s === 'up') return <Tag icon={<CheckCircleOutlined />} color="success">正常</Tag>
        return <Tag icon={<CloseCircleOutlined />} color="error">已下线</Tag>
      },
    },
    {
      title: '注册时间', dataIndex: 'register_time', key: 'register_time', width: 170,
      render: formatUnixTime,
    },
    {
      title: '操作', key: 'actions', width: 160,
      render: (_: unknown, record: GrpcInstance) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setPreviewJson(JSON.stringify(record, null, 2))}
            />
          </Tooltip>
          {isAdmin && record.status === 'up' && (
            <Popconfirm title="确认下线该实例？" onConfirm={() => handleUpdateStatus(record, 'down')}>
              <Tooltip title="下线">
                <Button size="small" danger icon={<StopOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
          {isAdmin && record.status !== 'up' && (
            <Popconfirm title="确认上线该实例？" onConfirm={() => handleUpdateStatus(record, 'up')}>
              <Tooltip title="上线">
                <Button size="small" type="primary" icon={<PlayCircleOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const collapseItems = groups.map((group) => ({
    key: group.service_name,
    label: (
      <Space>
        <span style={{ fontWeight: 500 }}>{group.service_name}</span>
        <Badge count={group.instance_count} style={{ backgroundColor: '#1677ff' }} />
        <Tag color="success">{group.healthy_count} 正常</Tag>
        {group.unhealthy_count > 0 && <Tag color="error">{group.unhealthy_count} 下线</Tag>}
      </Space>
    ),
    children: (
      <Table
        rowKey="instance_id"
        columns={instanceColumns}
        dataSource={group.instances}
        pagination={false}
        size="small"
      />
    ),
  }))

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ReloadOutlined />} onClick={() => fetchData()}>刷新</Button>
      </Space>

      {groups.length > 0 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Card><Statistic title="服务数" value={groups.length} /></Card>
          </Col>
          <Col span={8}>
            <Card><Statistic title="实例总数" value={totalInstances} /></Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="健康率"
                value={totalInstances > 0 ? ((totalHealthy / totalInstances) * 100).toFixed(1) : 0}
                suffix="%"
                valueStyle={{ color: totalHealthy === totalInstances ? '#3f8600' : '#cf1322' }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {loading ? (
        <Spin style={{ display: 'block', margin: '48px auto' }} />
      ) : groups.length > 0 ? (
        <Collapse items={collapseItems} defaultActiveKey={groups.map((g) => g.service_name)} />
      ) : (
        <Empty description="当前环境暂无注册 gRPC 服务" />
      )}

      <Modal
        title="实例详情"
        open={previewJson !== null}
        onCancel={() => setPreviewJson(null)}
        footer={null}
        width={700}
      >
        {previewJson !== null && (
          <MonacoEditor value={previewJson} language="json" readOnly height={400} />
        )}
      </Modal>
    </>
  )
}
```

- [ ] **Step 2: 验证前端编译**

Run: `cd /Users/dysodeng/project/go/etcd-manager/web && npx tsc --noEmit`
Expected: 编译成功，无类型错误

- [ ] **Step 3: Commit**

```bash
cd web && git add src/pages/grpc/index.tsx
git commit -m "feat: add gRPC service management page"
```

---

### Task 7: 注册前端路由和侧边栏菜单

**Files:**
- Modify: `web/src/App.tsx:11,40`
- Modify: `web/src/layouts/MainLayout.tsx:33-34`

- [ ] **Step 1: 在 App.tsx 中添加 gRPC 路由**

在 `web/src/App.tsx` 中：

1. 添加 import：
```typescript
import GrpcPage from '@/pages/grpc'
```

2. 在 `<Route path="gateway" .../>` 之后添加：
```tsx
<Route path="grpc" element={<GrpcPage />} />
```

- [ ] **Step 2: 在 MainLayout.tsx 中添加侧边栏菜单项**

在 `web/src/layouts/MainLayout.tsx` 中：

1. 在 import 中添加 `CloudServerOutlined`：
```typescript
import {
  DatabaseOutlined,
  SettingOutlined,
  ClusterOutlined,
  UserOutlined,
  AuditOutlined,
  LogoutOutlined,
  KeyOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ApiOutlined,
  CloudServerOutlined,
} from '@ant-design/icons'
```

2. 在 `menuItems` 数组中，`gateway` 项之后添加：
```typescript
  { key: '/grpc', icon: <CloudServerOutlined />, label: 'gRPC 服务' },
```

- [ ] **Step 3: 验证前端编译**

Run: `cd /Users/dysodeng/project/go/etcd-manager/web && npx tsc --noEmit`
Expected: 编译成功，无类型错误

- [ ] **Step 4: Commit**

```bash
cd web && git add src/App.tsx src/layouts/MainLayout.tsx
git commit -m "feat: add gRPC service route and sidebar menu"
```

