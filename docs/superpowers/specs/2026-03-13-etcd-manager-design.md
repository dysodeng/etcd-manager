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
| 数据库 | SQLite + GORM | 零运维，仓储模式抽象，可替换为 MySQL/PostgreSQL |
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
│   └── store/                   # 数据访问层（仓储模式）
│       ├── repository.go        # Repository 接口定义
│       ├── transaction.go       # 事务管理器接口
│       └── sqlite/              # SQLite 实现（可替换为 MySQL/PostgreSQL）
│           ├── repository.go    # 接口实现
│           └── transaction.go   # 事务管理器实现
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

### 权限模型

| 功能 | admin | viewer |
|------|-------|--------|
| KV 管理（读写） | ✅ | 只读 |
| 配置中心（读写） | ✅ | 只读 |
| 集群监控 | ✅ | ✅ |
| 用户管理 | ✅ | 不可见 |
| 审计日志 | ✅ | ✅ |
| 环境管理 | ✅ | 不可见 |
| 导入导出 | ✅ | 仅导出 |

viewer 可查看所有环境的配置数据，但不能修改。

### 初始用户

首次启动时自动创建默认 admin 用户（用户名 `admin`，密码 `admin123`），控制台输出提示信息，建议用户登录后立即修改密码。

### 错误码定义

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 10001 | 参数校验失败 |
| 10002 | 未授权（未登录或 Token 过期） |
| 10003 | 权限不足 |
| 20001 | Key 已存在 |
| 20002 | Key 不存在 |
| 20003 | 版本不存在 |
| 20004 | 环境已存在 |
| 20005 | 环境下存在配置，无法删除 |
| 30001 | etcd 连接失败 |
| 30002 | etcd 操作失败 |
| 40001 | 用户名已存在 |
| 40002 | 用户名或密码错误 |
| 50001 | 导入格式错误 |
| 50002 | 导入部分失败（返回失败明细） |

### 分页约定

- 默认 page=1，page_size=20
- page_size 上限 100

### 数据访问层设计（仓储模式）

数据访问层采用事务管理器 + 仓储（Repository）模式，Service 层依赖接口而非具体实现，后续切换数据库（MySQL/PostgreSQL）只需替换 DB 连接和实现层，不影响业务逻辑。

核心接口：

- `TransactionManager` — 事务管理器接口，提供 `WithTransaction(ctx, fn)` 方法
- `UserRepository` — 用户数据访问接口
- `EnvironmentRepository` — 环境数据访问接口
- `ConfigRevisionRepository` — 配置版本数据访问接口
- `AuditLogRepository` — 审计日志数据访问接口

Service 层通过构造函数注入 Repository 接口，不直接依赖 GORM 或任何具体数据库实现。

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
| POST | /api/v1/auth/logout | 登出（前端清除本地 Token，无需服务端处理） |
| GET | /api/v1/auth/profile | 获取当前用户信息 |

登出策略：JWT 无状态，登出由前端清除本地存储的 Token 实现。服务端不维护 Token 黑名单，Token 自然过期失效。

### KV 管理

KV 管理是通用的 etcd 操作入口，可操作 etcd 中的任意 key（包括配置中心管理的 key）。但通过 KV 管理修改配置中心前缀下的 key 时，不会自动记录版本历史。建议用户通过配置中心页面管理带环境前缀的配置，KV 管理用于调试或管理非配置类 key。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/kv?prefix=xxx&limit=50 | 按前缀列出 KV |
| GET | /api/v1/kv?key=xxx | 获取单个 KV |
| POST | /api/v1/kv | 创建 KV |
| PUT | /api/v1/kv | 更新 KV |
| DELETE | /api/v1/kv?key=xxx | 删除 KV |

key 通过 query 参数传递（而非路径参数），因为 etcd key 包含 `/` 字符，放在路径中会导致路由匹配问题。

### 配置中心

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/environments | 环境列表 |
| POST | /api/v1/environments | 创建环境 |
| PUT | /api/v1/environments/:id | 更新环境 |
| DELETE | /api/v1/environments/:id | 删除环境（需确认无配置数据） |
| GET | /api/v1/configs?env=dev&prefix=app/ | 按环境查配置 |
| POST | /api/v1/configs | 创建配置（自动记录版本），请求体含 `env` 字段 |
| PUT | /api/v1/configs | 更新配置（自动记录版本），请求体含 `env` 和 `key` |
| DELETE | /api/v1/configs?env=dev&key=app/db_host | 删除配置 |
| GET | /api/v1/configs/revisions?env=dev&key=app/db_host | 查看配置变更历史 |
| POST | /api/v1/configs/rollback | 回滚到指定版本，请求体 `{ "env": "dev", "key": "app/db_host", "revision_id": 123 }` |
| POST | /api/v1/configs/import?env=dev&dry_run=true | 导入配置（JSON/YAML），key 已存在则覆盖，`dry_run=true` 时仅预览不执行 |
| GET | /api/v1/configs/export?env=dev&format=json | 导出配置，format 支持 json/yaml，默认 json |

配置中心的 key 是去掉环境前缀后的短 key（如 `app/db_host`），服务端根据 `env` 参数自动拼接完整 etcd key（如 `/config/dev/app/db_host`）。config_revisions 表中的 key 字段同样存储短 key，通过 environment_id 关联环境。

### 实时推送（SSE）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/watch?prefix=xxx | SSE 长连接，推送 KV 变更事件 |

SSE 事件格式：

```
event: kv_change
data: {"type": "PUT|DELETE", "key": "/config/dev/app/db_host", "value": "new_value", "revision": 12345}
id: 12345
```

- 需要 JWT 认证（通过 query 参数 `token=xxx` 传递）
- 支持 `Last-Event-ID` 头实现断线重连，服务端从指定 revision 开始重放。若 revision 已被 etcd compaction 回收，返回当前最新数据并附带 `compacted` 标记
- 连接空闲超时 30 分钟自动断开，客户端自动重连

### 集群信息

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/cluster/status | 集群健康状态、成员列表 |
| GET | /api/v1/cluster/metrics | 基础指标 |

集群 metrics 返回字段：leader ID、DB 大小、成员数量、各成员健康状态、etcd 版本。

### 用户管理（admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/users | 用户列表 |
| POST | /api/v1/users | 创建用户 |
| PUT | /api/v1/users/:id | 更新用户 |
| DELETE | /api/v1/users/:id | 删除用户 |
| GET | /api/v1/audit-logs | 审计日志查询 |

审计日志筛选参数：`user_id`、`action`、`resource_type`、`start_time`、`end_time`，均为可选，支持分页。

### 个人设置

| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | /api/v1/auth/password | 修改当前用户密码，请求体 `{ "old_password": "xxx", "new_password": "xxx" }` |

所有角色均可修改自己的密码，无需 admin 权限。

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
| 状态管理 | Zustand |;
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
  secret: "your-secret-key"  # 生产环境建议通过环境变量 JWT_SECRET 注入
  expire_hours: 24

log:
  level: "info"
```
