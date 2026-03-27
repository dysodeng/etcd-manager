# RBAC 权限增强设计

## 概述

将现有的二级角色模型（admin/viewer）升级为灵活的自定义角色权限系统。每个角色可配置所管理的环境和功能模块的读/写权限。

## 核心规则

- 系统保留唯一超级管理员（`is_super = true`），拥有全部权限，不受角色约束
- 超级管理员不分配角色（`role_id = NULL`）
- 其他用户必须分配一个自定义角色，一个用户只能分配一个角色
- 用户只能看到和操作其角色被授权的环境
- 每个功能模块独立配置读/写权限

---

## 数据模型

### 用户表变更（`users`）

```sql
-- 移除: role VARCHAR(16) DEFAULT 'viewer'
-- 新增:
is_super  BOOLEAN DEFAULT FALSE   -- 超级管理员标识，全局唯一一个为 TRUE
role_id   UUID NULL               -- 关联角色，超级管理员此字段为 NULL
```

### 新增：角色表（`roles`）

```sql
id          UUID PRIMARY KEY
name        VARCHAR(64) UNIQUE NOT NULL  -- 角色名称，如"运维组"、"开发组"
description VARCHAR(255)                 -- 角色描述
created_at  TIMESTAMP
updated_at  TIMESTAMP
```

### 新增：角色模块权限表（`role_permissions`）

```sql
id        UUID PRIMARY KEY
role_id   UUID NOT NULL REFERENCES roles(id)
module    VARCHAR(32) NOT NULL  -- 模块标识
can_read  BOOLEAN DEFAULT FALSE
can_write BOOLEAN DEFAULT FALSE
UNIQUE(role_id, module)
```

模块标识枚举：

| 标识 | 说明 |
|------|------|
| `kv` | KV 管理 |
| `config` | 配置中心 |
| `gateway` | 网关服务 |
| `grpc` | gRPC 服务 |
| `users` | 用户管理 |
| `environments` | 环境管理 |
| `audit_logs` | 审计日志 |
| `cluster` | 集群信息 |

### 新增：角色环境关联表（`role_environments`）

```sql
id              UUID PRIMARY KEY
role_id         UUID NOT NULL REFERENCES roles(id)
environment_id  UUID NOT NULL REFERENCES environments(id)
UNIQUE(role_id, environment_id)
```

### 数据完整性规则

- `is_super = true` 的用户：跳过所有权限检查，`role_id` 为 NULL
- 普通用户必须有 `role_id`，否则无任何权限
- 删除角色前必须先解除所有用户绑定
- 删除环境时级联删除 `role_environments` 中的关联记录

---

## API 设计

### 角色管理 API（`/roles`，仅超级管理员）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/roles` | 角色列表（分页） |
| GET | `/roles/:id` | 角色详情（含权限和环境配置） |
| POST | `/roles` | 创建角色 |
| PUT | `/roles/:id` | 更新角色（含权限和环境） |
| DELETE | `/roles/:id` | 删除角色（需先解绑用户） |

创建/更新角色请求体：

```json
{
  "name": "运维组",
  "description": "负责线上环境运维",
  "permissions": [
    {"module": "kv", "can_read": true, "can_write": true},
    {"module": "config", "can_read": true, "can_write": false},
    {"module": "users", "can_read": false, "can_write": false}
  ],
  "environment_ids": ["uuid-1", "uuid-2"]
}
```

### 用户管理 API 变更

- 创建用户：`role` 字段改为 `role_id`
- 更新用户：可更新 `role_id`
- 用户列表：返回关联的角色名称
- 超级管理员转移：新增 `PUT /users/:id/transfer-super`（仅超级管理员可操作）

### 权限中间件

替换当前的 `RequireAdmin()` 和 `RequireAdminForWrite()`：

1. **`RequireSuper()`** — 仅超级管理员可访问（角色管理、超管转移）
2. **`RequirePermission(module, action)`** — 检查角色的模块读/写权限
   - `action = "read"` → 检查 `can_read`
   - `action = "write"` → 检查 `can_write`（写权限隐含读权限）
3. **`FilterEnvironments()`** — 环境列表过滤中间件

中间件逻辑：
```
if user.is_super → 放行
if user.role_id == nil → 403
检查 role_permissions 中 module+action 权限 → 放行/拒绝
```

### 环境过滤逻辑

所有涉及环境的接口（KV、Config、Gateway、gRPC 等）：
- 超级管理员：返回所有环境
- 普通用户：仅返回 `role_environments` 中关联的环境
- 操作未授权环境 → 403

### 路由权限分配

```
仅超级管理员:
  /roles/*                     → RequireSuper()
  PUT /users/:id/transfer-super → RequireSuper()

按模块权限控制:
  /kv/*          → RequirePermission("kv", read/write)
  /configs/*     → RequirePermission("config", read/write)
  /gateway/*     → RequirePermission("gateway", read/write)
  /grpc/*        → RequirePermission("grpc", read/write)
  /users/*       → RequirePermission("users", read/write)
  /environments  → RequirePermission("environments", read) + FilterEnvironments()
  /audit-logs    → RequirePermission("audit_logs", read)
  /cluster/*     → RequirePermission("cluster", read)
  /watch         → RequirePermission("kv", read) + FilterEnvironments()
```

---

## 前端变更

### 新增：角色管理页面

- 路由：`/roles`，侧边栏新增"角色管理"菜单项，仅超级管理员可见
- 角色列表页：显示角色名、描述、关联用户数、操作按钮
- 角色编辑弹窗：
  - 基本信息：名称、描述
  - 环境选择：多选框，列出所有环境，勾选授权的环境
  - 模块权限：表格形式，每行一个模块，两列复选框（读/写）
  - 写权限自动勾选读权限

### 用户管理页面变更

- 创建/编辑用户时，角色选择改为下拉选择自定义角色
- 超级管理员用户显示"超级管理员"标签，不显示角色选择器
- 新增"转移超管"按钮（仅超级管理员在自己的用户详情中可见）

### 布局与环境切换

- 环境下拉框：后端返回已过滤的列表，前端直接使用
- 侧边栏菜单：根据角色模块权限动态显示/隐藏
  - 无读权限 → 隐藏菜单
  - 有读无写 → 显示菜单但禁用写操作按钮
- 超级管理员看到全部菜单和全部环境

### Auth Store 变更

```typescript
interface UserProfile {
  id: string
  username: string
  is_super: boolean
  role: {
    id: string
    name: string
    permissions: { module: string; can_read: boolean; can_write: boolean }[]
    environment_ids: string[]
  } | null  // 超级管理员为 null
}
```

工具函数：
- `canRead(module: string): boolean` — 超管返回 true，否则检查 permissions
- `canWrite(module: string): boolean` — 超管返回 true，否则检查 permissions
- `isSuper(): boolean` — 检查 is_super 字段

---

## 数据迁移

1. 用户表添加 `is_super`、`role_id` 字段
2. 将现有 `admin` 用户标记为 `is_super = true`
3. 创建默认角色"管理员"（所有模块读写权限 + 所有环境）和"观察者"（所有模块只读 + 所有环境）
4. 将现有 `role = "admin"` 的非超管用户分配"管理员"角色
5. 将现有 `role = "viewer"` 用户分配"观察者"角色
6. 移除用户表 `role` 字段
