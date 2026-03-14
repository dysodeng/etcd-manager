# 网关服务管理 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增网关服务管理模块，从 etcd 读取服务注册信息，按服务名分组展示实例列表，支持查看详情和下线实例。

**Architecture:** 服务实例数据存储在 etcd 中，key 格式为 `/{project}/services/{env}/{service_name}/{instance_id}`，value 为 JSON。后端新增 GatewayService 从 etcd 读取并解析，前端新增页面按服务名分组展示，支持实例详情查看和下线操作。

**Tech Stack:** Go/Gin/etcd (backend), React/TypeScript/Ant Design (frontend)

---

## 文件变更清单

### 后端新建
- `internal/service/gateway_service.go` — 网关服务层，从 etcd 读取/解析/删除服务实例
- `internal/handler/gateway.go` — HTTP handler，提供 API 端点

### 后端修改
- `internal/handler/router.go` — 注册新路由
- `internal/handler/router.go:9-17` — Handlers 结构体加 Gateway 字段
- `cmd/server/main.go` — 初始化 GatewayService 和 GatewayHandler

### 前端新建
- `web/src/api/gateway.ts` — API 客户端
- `web/src/pages/gateway/index.tsx` — 网关服务管理页面

### 前端修改
- `web/src/types/index.ts` — 新增类型定义
- `web/src/App.tsx` — 新增路由
- `web/src/layouts/MainLayout.tsx` — 新增菜单项

---

## Chunk 1: 后端实现

### Task 1: 创建 GatewayService

**Files:**
- Create: `internal/service/gateway_service.go`

- [ ] **Step 1: 创建 gateway_service.go**

```go
package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/dysodeng/config-center/internal/etcd"
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
		if inst.Status == "healthy" {
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

// DeregisterInstance 下线指定实例（删除 etcd key）
func (s *GatewayService) DeregisterInstance(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := s.etcdClient.Delete(ctx, key)
	return err
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/dysodeng/project/go/config-center && go build ./internal/service/`
Expected: 无错误

---

### Task 2: 创建 GatewayHandler

**Files:**
- Create: `internal/handler/gateway.go`

- [ ] **Step 1: 创建 gateway.go**

```go
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/service"
)

type GatewayHandler struct {
	gatewaySvc *service.GatewayService
	auditSvc   *service.AuditService
}

func NewGatewayHandler(gatewaySvc *service.GatewayService, auditSvc *service.AuditService) *GatewayHandler {
	return &GatewayHandler{gatewaySvc: gatewaySvc, auditSvc: auditSvc}
}

// List 列出所有服务（按服务名分组）
func (h *GatewayHandler) List(c *gin.Context) {
	prefix := c.Query("prefix")
	if prefix == "" {
		Fail(c, CodeBadRequest, "prefix is required")
		return
	}

	groups, err := h.gatewaySvc.ListServices(c.Request.Context(), prefix)
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, groups)
}

// Deregister 下线实例
func (h *GatewayHandler) Deregister(c *gin.Context) {
	var req struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeBadRequest, "key is required")
		return
	}

	if err := h.gatewaySvc.DeregisterInstance(c.Request.Context(), req.Key); err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}

	userID, _ := getUserID(c)
	h.auditSvc.Log(c.Request.Context(), userID, "deregister", "service_instance", req.Key, "", c.ClientIP())

	OK(c, nil)
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/dysodeng/project/go/config-center && go build ./internal/handler/`
Expected: 可能报错（Handlers 结构体还没加 Gateway 字段），下一步修复

---

### Task 3: 注册路由和接线

**Files:**
- Modify: `internal/handler/router.go:9-17` — Handlers 加 Gateway
- Modify: `internal/handler/router.go` — 注册路由
- Modify: `cmd/server/main.go` — 初始化

- [ ] **Step 1: 修改 router.go — Handlers 结构体加 Gateway**

在 `Handlers` 结构体中添加：
```go
type Handlers struct {
	Auth         *AuthHandler
	KV           *KVHandler
	ConfigCenter *ConfigCenterHandler
	Watch        *WatchHandler
	Cluster      *ClusterHandler
	User         *UserHandler
	Audit        *AuditHandler
	Gateway      *GatewayHandler  // 新增
}
```

- [ ] **Step 2: 修改 router.go — 注册路由**

在 `RegisterRoutes` 函数中，`auth.GET("/audit-logs", h.Audit.List)` 之后添加：
```go
		// 网关服务管理（viewer 只读，admin 可下线）
		gateway := auth.Group("/gateway", middleware.RequireAdminForWrite())
		{
			gateway.GET("", h.Gateway.List)
			gateway.DELETE("", h.Gateway.Deregister)
		}
```

- [ ] **Step 3: 修改 main.go — 初始化 GatewayService 和 Handler**

在 `clusterSvc := ...` 之后添加：
```go
	gatewaySvc := service.NewGatewayService(etcdClient)
```

在 `handlers := &handler.Handlers{` 中添加：
```go
		Gateway:      handler.NewGatewayHandler(gatewaySvc, auditSvc),
```

- [ ] **Step 4: 验证编译**

Run: `cd /Users/dysodeng/project/go/config-center && go build ./...`
Expected: 编译通过

- [ ] **Step 5: Commit**

```bash
git add internal/service/gateway_service.go internal/handler/gateway.go internal/handler/router.go cmd/server/main.go
git commit -m "feat: add gateway service management backend"
```

---

## Chunk 2: 前端实现

### Task 4: 新增前端类型和 API

**Files:**
- Modify: `web/src/types/index.ts` — 新增类型
- Create: `web/src/api/gateway.ts` — API 客户端

- [ ] **Step 1: 在 types/index.ts 末尾添加类型**

```typescript
// Gateway
export interface ServiceInstance {
  id: string
  service_name: string
  host: string
  port: number
  weight: number
  version: string
  status: 'healthy' | 'unhealthy' | string
  registered_at: string
  metadata: Record<string, string>
}

export interface ServiceGroup {
  service_name: string
  instance_count: number
  healthy_count: number
  unhealthy_count: number
  instances: ServiceInstance[]
}
```

- [ ] **Step 2: 创建 api/gateway.ts**

```typescript
import client, { request } from './client'
import type { ServiceGroup } from '@/types'

export const gatewayApi = {
  list: (prefix: string) =>
    request<ServiceGroup[]>(client.get('/gateway', { params: { prefix } })),
  deregister: (key: string) =>
    request<null>(client.delete('/gateway', { data: { key } })),
}
```

---

### Task 5: 创建网关服务管理页面

**Files:**
- Create: `web/src/pages/gateway/index.tsx`

- [ ] **Step 1: 创建页面**

页面功能：
- 顶部输入框输入 etcd key 前缀（如 `/ai-adp/services/dev/`），点搜索加载
- 左侧服务列表（Collapse/卡片），显示服务名、实例数、健康数
- 点击服务展开实例表格：ID、Host:Port、版本、权重、状态 Tag、注册时间、操作（查看详情/下线）
- 查看详情弹窗：用 MonacoEditor 只读展示完整 JSON（含 metadata）
- 下线按钮需 admin 权限，二次确认

```typescript
import { useEffect, useState } from 'react'
import {
  Card, Table, Button, Input, Space, Tag, Modal, Popconfirm,
  Statistic, Row, Col, Collapse, Badge, Tooltip, Empty, Spin, message,
} from 'antd'
import {
  SearchOutlined, ReloadOutlined, EyeOutlined,
  StopOutlined, CheckCircleOutlined, CloseCircleOutlined,
} from '@ant-design/icons'
import type { ServiceGroup, ServiceInstance } from '@/types'
import { gatewayApi } from '@/api/gateway'
import { useAuthStore } from '@/stores/auth'
import { useEnvironmentStore } from '@/stores/environment'
import MonacoEditor from '@/components/MonacoEditor'
import { formatTime } from '@/utils'

export default function GatewayPage() {
  const currentEnv = useEnvironmentStore((s) => s.current)
  const isAdmin = useAuthStore((s) => s.user?.role === 'admin')

  const [groups, setGroups] = useState<ServiceGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [prefix, setPrefix] = useState('')
  const [previewJson, setPreviewJson] = useState<string | null>(null)

  // 环境切换时自动拼接前缀
  useEffect(() => {
    if (currentEnv?.key_prefix) {
      const base = currentEnv.key_prefix.endsWith('/')
        ? currentEnv.key_prefix
        : currentEnv.key_prefix + '/'
      setPrefix(base + 'services/')
    }
  }, [currentEnv])

  const fetchData = async () => {
    if (!prefix) return
    setLoading(true)
    try {
      const data = await gatewayApi.list(prefix)
      setGroups(data ?? [])
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }

  const handleDeregister = async (instance: ServiceInstance) => {
    // 根据前缀和实例信息拼出完整 key
    const key = prefix + instance.service_name + '/' + instance.id
    try {
      await gatewayApi.deregister(key)
      message.success('实例已下线')
      fetchData()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '操作失败')
    }
  }

  const totalInstances = groups.reduce((sum, g) => sum + g.instance_count, 0)
  const totalHealthy = groups.reduce((sum, g) => sum + g.healthy_count, 0)

  const instanceColumns = [
    {
      title: 'ID', dataIndex: 'id', key: 'id', width: 280,
      render: (id: string) => (
        <Tooltip title={id}>
          <span style={{ fontFamily: 'monospace' }}>{id.slice(0, 8)}...</span>
        </Tooltip>
      ),
    },
    {
      title: '地址', key: 'address',
      render: (_: unknown, r: ServiceInstance) => (
        <span style={{ fontFamily: 'monospace' }}>{r.host}:{r.port}</span>
      ),
    },
    { title: '版本', dataIndex: 'version', key: 'version', width: 100 },
    { title: '权重', dataIndex: 'weight', key: 'weight', width: 80 },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s: string) =>
        s === 'healthy'
          ? <Tag icon={<CheckCircleOutlined />} color="success">健康</Tag>
          : <Tag icon={<CloseCircleOutlined />} color="error">{s}</Tag>,
    },
    {
      title: '注册时间', dataIndex: 'registered_at', key: 'registered_at', width: 170,
      render: formatTime,
    },
    {
      title: '操作', key: 'actions', width: 120,
      render: (_: unknown, record: ServiceInstance) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setPreviewJson(JSON.stringify(record, null, 2))}
            />
          </Tooltip>
          {isAdmin && (
            <Popconfirm title="确认下线该实例？" onConfirm={() => handleDeregister(record)}>
              <Tooltip title="下线">
                <Button size="small" danger icon={<StopOutlined />} />
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
        <Tag color="success">{group.healthy_count} 健康</Tag>
        {group.unhealthy_count > 0 && <Tag color="error">{group.unhealthy_count} 异常</Tag>}
      </Space>
    ),
    children: (
      <Table
        rowKey="id"
        columns={instanceColumns}
        dataSource={group.instances}
        pagination={false}
        size="small"
      />
    ),
  }))

  return (
    <>
      <Space style={{ marginBottom: 16 }} wrap>
        <Input
          prefix={<SearchOutlined />}
          placeholder="服务 Key 前缀"
          value={prefix}
          onChange={(e) => setPrefix(e.target.value)}
          onPressEnter={() => fetchData()}
          style={{ width: 400 }}
        />
        <Button type="primary" icon={<SearchOutlined />} onClick={() => fetchData()}>搜索</Button>
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
        <Empty description="暂无服务数据，请输入前缀搜索" />
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

---

### Task 6: 注册路由和菜单

**Files:**
- Modify: `web/src/App.tsx` — 新增路由
- Modify: `web/src/layouts/MainLayout.tsx` — 新增菜单项

- [ ] **Step 1: 修改 App.tsx**

添加 import：
```typescript
import GatewayPage from '@/pages/gateway'
```

在 `<Route path="audit" .../>` 之后添加：
```typescript
<Route path="gateway" element={<GatewayPage />} />
```

- [ ] **Step 2: 修改 MainLayout.tsx**

在 menuItems 数组中，`配置中心` 之后添加：
```typescript
{ key: '/gateway', icon: <ApiOutlined />, label: '网关服务' },
```

并在 import 中添加 `ApiOutlined`。

- [ ] **Step 3: 验证编译**

Run: `cd /Users/dysodeng/project/go/config-center/web && npx tsc --noEmit`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add web/src/
git commit -m "feat: add gateway service management frontend"
```
