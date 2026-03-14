# etcd-manager

一个功能完善的 etcd 集群可视化管理平台，提供通用 KV 管理、配置中心、网关服务管理、集群监控、用户权限控制和审计日志等功能。

## 功能特性

- **通用 KV 管理** — 支持 etcd 键值对的增删改查，前缀搜索与排序
- **配置中心** — 多环境隔离（dev/staging/prod），版本历史追踪与回滚，JSON/YAML 格式导入导出
- **实时监控** — 基于 SSE 的 KV 变更实时推送，集群健康状态与成员信息展示
- **网关服务管理** — 服务实例注册发现，健康状态追踪，Lease 安全状态更新
- **用户与权限** — 基于 JWT 的认证，角色权限控制（admin/viewer）
- **审计日志** — 全操作审计记录，支持按用户、操作类型、时间范围筛选

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25+, Gin, GORM, etcd client v3 |
| 前端 | React 18, TypeScript, Vite, Ant Design, Monaco Editor |
| 数据库 | SQLite（默认）/ PostgreSQL |
| 认证 | JWT + bcrypt |

## 快速开始

### 前置条件

- Go 1.25+
- Node.js 18+
- 运行中的 etcd 集群

### 后端

```bash
# 克隆项目
git clone https://github.com/dysodeng/etcd-manager.git
cd etcd-manager

# 编译
go build -o etcd-manager ./cmd/server

# 运行（默认读取 configs/config.yaml）
./etcd-manager
```

### 前端

```bash
cd web

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
```

### 默认账号

- 用户名：`admin`
- 密码：`admin123`

> 首次启动自动创建，请及时修改密码。

## 配置说明

编辑 `configs/config.yaml`：

```yaml
server:
  port: 8080

etcd:
  endpoints:
    - "127.0.0.1:2379"
  username: ""
  password: ""
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
    ca_file: ""

database:
  driver: "sqlite"  # sqlite 或 postgres
  path: "./data/data.db"
  # PostgreSQL 配置（driver 为 postgres 时使用）
  # dsn: "host=localhost port=5432 user=postgres password=postgres dbname=config_center sslmode=disable"

jwt:
  secret: "change-me-in-production"  # 生产环境请修改，可通过 JWT_SECRET 环境变量覆盖
  expire_hours: 24

log:
  level: "info"
```

## 项目结构

```
├── cmd/server/          # 应用入口
├── internal/
│   ├── config/          # 配置加载
│   ├── domain/          # 领域模型与仓储接口
│   ├── handler/         # HTTP 路由处理
│   ├── middleware/       # JWT 认证、CORS、角色权限中间件
│   ├── service/         # 业务逻辑层
│   ├── etcd/            # etcd 客户端封装
│   ├── store/           # 数据访问层（SQLite / PostgreSQL）
│   ├── seed/            # 数据库初始化
│   └── response/        # 响应格式化
├── web/                 # React 前端
├── configs/             # 配置文件
└── data/                # SQLite 数据库文件
```

## API 概览

所有接口前缀为 `/api/v1`，需 JWT 认证（登录接口除外）。

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | `POST /auth/login` | 登录获取 Token |
| KV 管理 | `GET/POST/PUT/DELETE /kv` | 键值对 CRUD |
| 配置中心 | `GET/POST /configs` | 配置管理 |
| 配置中心 | `GET /configs/revisions` | 版本历史 |
| 配置中心 | `POST /configs/rollback` | 版本回滚 |
| 配置中心 | `POST /configs/import` | 导入配置 |
| 配置中心 | `GET /configs/export` | 导出配置 |
| 环境 | `GET/POST /environments` | 环境管理 |
| 集群 | `GET /cluster/status` | 集群状态 |
| 集群 | `GET /cluster/metrics` | 集群指标 |
| 实时监听 | `GET /watch` | SSE 变更推送 |
| 网关 | `GET /gateway` | 服务列表 |
| 用户 | `GET/POST/PUT/DELETE /users` | 用户管理（仅 admin） |
| 审计 | `GET /audit-logs` | 审计日志查询 |

## License

[MIT](LICENSE)
