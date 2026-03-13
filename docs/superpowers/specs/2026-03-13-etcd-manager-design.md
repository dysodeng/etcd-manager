# etcd 管理服务 + 配置中心 设计文档

## 概述

构建一个基于 etcd 的管理服务，兼具通用 etcd KV 管理和配置中心高级功能。提供 Web 端操作界面，面向开发/运维团队日常使用。

## 需求摘要

- 通用 etcd KV 管理（增删改查、前缀搜索）
- 配置中心能力：版本管理、变更通知（Watch）、环境隔离、导入导出
- 单集群模式
- 用户名密码认证，角色区分（admin/viewer）
- 前后端分离部署

## 架构

### 技术选型

| 层级 | 技术 | 说明 |
|------|------|------|
| 后端框架 | Go + Gin | 轻量高性能 |
| 数据库 | SQLite + GORM | 零运维，存储版本历史/用户/审计 |
| etcd 客户端 | etcd client v3 | KV 操作、Watch、Lease |
| 认证 | JWT | 无状态，前后端分离友好 |
| 实时推送 | SSE | 比 WebSocket 简单，天然支持重连 |
| 前端框架 | React + Ant Design | 组件库强大 |
| 构建工具 | Vite + TypeScript | 快速构建，类型安全 |
| 状态管理 | Zustand | 轻量 |
| HTTP 客户端 | Axios | |
| 代码编辑器 | Monaco Editor | 配置值编辑 |

### 系统架构图

```
┌─────────────────────────────────────────────────┐
│              React + Ant Design                  │
│  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌──────┐ │
│  │ KV 管理   │ │ 配置中心  │ │集群监控 │ │用户  │ │
│  └──────────┘ └──────────┘ └────────┘ └──────┘ │
└──────────────────┬──────────────┬───────────────┘
            REST API│          SSE│
                    ▼              ▼
┌─────────────────────────────────────────────────┐
│              Go 后端服务 (Gin)                    │
│  ┌───────────┐ ┌───────────┐ ┌──────────────┐  │
│  │ API Layer  │→│ Service层  │ │ Auth中间件    │  │
│  └───────────┘ └─────┬─────┘ │ (JWT)        │  │
│                      │        └──────────────┘  │
│            ┌─────────┼─────────┐                │
│            ▼                   ▼                │
│  ┌────────────────┐ ┌──────────────────┐        │
│  │ etcd Client v3  │ │ SQLite (GORM)    │        │
│  └───────┬────────┘ └──────────────────┘        │
└──────────┼──────────────────────────────────────┘
           ▼
┌─────────────────┐
│  etcd Cluster    │
└─────────────────┘
```

## 后端设计

### 项目目录结构

```
config-center/
├── cmd/
│   └── server/
│       └── main.go              # 入口
├── internal/
│   ├── config/                  # 应用配置（读取 YAML）
│   ├── handler/                 # HTTP Handler
│   │   ├── auth.go
│   │   ├── kv.go
│   │   ├── config_center.go
│   │   ├── watch.go
│   │   ├── cluster.go
│   │   └── user.go
│   ├── service/                 # 业务逻辑层
│   ├── model/                   # 数据模型（GORM）
│   ├── middleware/              # Gin 中间件（JWT、CORS、日志）
│   ├── etcd/                    # etcd 客户端封装
│   └── store/                   # SQLite 数据访问层
├── configs/
│   └── config.yaml
├── go.mod
└── go.sum
```

### 数据模型（SQLite）

#### users 用户表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| username | string | 用户名，唯一 |
| password_hash | string | bcrypt 哈希 |
| role | string | admin / viewer |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |

#### environments 环境表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| name | string | 环境名（dev/staging/prod） |
| key_prefix | string | etcd key 前缀（/config/dev/） |
| description | string | 描述 |
| sort_order | int | 排序 |

#### config_revisions 配置版本表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| environment_id | uint | 关联环境 |
| key | string | 配置 key |
| value | text | 当前值 |
| prev_value | text | 变更前的值 |
| etcd_revision | int64 | etcd revision |
| action | string | create / update / delete |
| operator | uint | 操作人 user_id |
| comment | string | 变更备注 |
| created_at | time | 创建时间 |

#### audit_logs 审计日志表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 操作人 |
| action | string | 操作类型 |
| resource_type | string | 资源类型 |
| resource_key | string | 资源标识 |
| detail | json | 详细信息 |
| ip | string | 客户端 IP |
| created_at | time | 创建时间 |

### 环境隔离策略

通过 etcd key 前缀实现环境隔离：

```
/config/dev/app/database_url     → 开发环境
/config/staging/app/database_url → 预发环境
/config/prod/app/database_url    → 生产环境
```

前端切换环境 = 切换 key 前缀。

## API 设计

所有接口统一前缀 `/api/v1`，JWT 认证（登录接口除外）。

### 统一响应格式

```json
// 成功
{ "code": 0, "message": "ok", "data": { ... } }

// 失败
{ "code": 10001, "message": "key already exists", "data": null }

// 分页
{ "code": 0, "message": "ok", "data": { "list": [...], "total": 100, "page": 1, "page_size": 20 } }
```

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 登录，返回 JWT |
| POST | /api/v1/auth/logout | 登出 |
| GET | /api/v1/auth/profile | 获取当前用户信息 |

### KV 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/kv?prefix=xxx&limit=50 | 按前缀列出 KV |
| GET | /api/v1/kv/:key | 获取单个 KV |
| POST | /api/v1/kv | 创建 KV |
| PUT | /api/v1/kv | 更新 KV |
| DELETE | /api/v1/kv/:key | 删除 KV |

### 配置中心

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/environments | 环境列表 |
| POST | /api/v1/environments | 创建环境 |
| GET | /api/v1/configs?env=dev&prefix=app/ | 按环境查配置 |
| POST | /api/v1/configs | 创建配置（自动记录版本） |
| PUT | /api/v1/configs | 更新配置（自动记录版本） |
| DELETE | /api/v1/configs/:key | 删除配置 |
| GET | /api/v1/configs/:key/revisions | 查看配置变更历史 |
| POST | /api/v1/configs/:key/rollback | 回滚到指定版本 |
| POST | /api/v1/configs/import | 导入配置（JSON/YAML） |
| GET | /api/v1/configs/export?env=dev | 导出配置 |

### 实时推送

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/watch?prefix=xxx | SSE 长连接，推送 KV 变更事件 |

### 集群信息

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/cluster/status | 集群健康状态、成员列表 |
| GET | /api/v1/cluster/metrics | 基础指标（leader、DB size 等） |

### 用户管理（admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/users | 用户列表 |
| POST | /api/v1/users | 创建用户 |
| PUT | /api/v1/users/:id | 更新用户 |
| DELETE | /api/v1/users/:id | 删除用户 |
| GET | /api/v1/audit-logs | 审计日志查询 |

## 前端设计

### 页面布局

左侧固定导航 + 顶栏（环境切换 + 用户信息） + 内容区。

### 页面清单

| 页面 | 说明 |
|------|------|
| 登录页 | 用户名密码表单，居中卡片布局 |
| KV 管理 | 列表视图，搜索过滤，CRUD 弹窗 |
| 配置中心 | 环境切换 Tab，配置列表，版本历史抽屉 |
| 集群监控 | 成员列表、Leader 状态、DB 大小、健康检查 |
| 用户管理 | 用户 CRUD 表格，角色分配（admin 可见） |
| 审计日志 | 操作记录时间线，按用户/操作类型筛选 |

### 前端技术栈

| 项目 | 选型 |
|------|------|
| 构建工具 | Vite |
| 语言 | TypeScript |
| 路由 | React Router v6 |
| 状态管理 | Zustand |
| HTTP 客户端 | Axios |
| 代码编辑器 | Monaco Editor |

### 前端目录结构

```
web/
├── src/
│   ├── api/                # API 请求封装
│   ├── components/         # 通用组件
│   ├── layouts/            # 布局组件
│   ├── pages/              # 页面
│   │   ├── login/
│   │   ├── kv/
│   │   ├── config/
│   │   ├── cluster/
│   │   ├── users/
│   │   └── audit/
│   ├── stores/             # Zustand stores
│   ├── hooks/              # 自定义 hooks
│   ├── utils/              # 工具函数
│   ├── types/              # TypeScript 类型定义
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## 部署

前后端分离部署：
- 后端：Go 二进制 + config.yaml + SQLite 文件
- 前端：Vite 构建产物，Nginx 托管，反向代理 API 到后端

### 配置文件示例（config.yaml）

```yaml
server:
  port: 8080

etcd:
  endpoints:
    - "localhost:2379"
  username: ""
  password: ""
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
    ca_file: ""

database:
  path: "./data/config-center.db"

jwt:
  secret: "your-secret-key"
  expire_hours: 24

log:
  level: "info"
```
