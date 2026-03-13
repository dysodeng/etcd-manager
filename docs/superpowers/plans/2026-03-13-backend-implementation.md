# etcd 管理服务 — 后端实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 etcd 管理服务的 Go 后端，提供 RESTful API 供前端调用。

**Architecture:** Go + Gin 框架，仓储模式 + 事务管理器抽象数据访问层（SQLite/GORM 实现），etcd client v3 封装 KV/Watch 操作，JWT 认证，SSE 实时推送。

**Tech Stack:** Go 1.25, Gin, GORM, SQLite, etcd client v3, JWT (golang-jwt), bcrypt, Viper (配置), SSE

**Spec:** `docs/superpowers/specs/2026-03-13-etcd-manager-design.md`

---

## File Structure

```
config-center/
├── cmd/server/main.go                          # 入口：初始化配置、DB、etcd、路由，启动服务
├── configs/config.yaml                         # 默认配置文件
├── internal/
│   ├── config/config.go                        # 应用配置结构体 + Viper 加载
│   ├── model/
│   │   ├── user.go                             # User GORM 模型
│   │   ├── environment.go                      # Environment GORM 模型
│   │   ├── config_revision.go                  # ConfigRevision GORM 模型
│   │   └── audit_log.go                        # AuditLog GORM 模型
│   ├── store/
│   │   ├── repository.go                       # Repository 接口定义（UserRepo, EnvRepo, RevisionRepo, AuditRepo）
│   │   ├── transaction.go                      # TransactionManager 接口
│   │   └── sqlite/
│   │       ├── db.go                           # SQLite 连接初始化 + AutoMigrate
│   │       ├── transaction.go                  # TransactionManager 实现
│   │       ├── user_repository.go              # UserRepository 实现
│   │       ├── environment_repository.go       # EnvironmentRepository 实现
│   │       ├── config_revision_repository.go   # ConfigRevisionRepository 实现
│   │       └── audit_log_repository.go         # AuditLogRepository 实现
│   ├── etcd/client.go                          # etcd 客户端封装（连接、KV、Watch、集群信息）
│   ├── service/
│   │   ├── auth_service.go                     # 登录、JWT 生成/验证、密码修改
│   │   ├── kv_service.go                       # 通用 KV CRUD
│   │   ├── config_service.go                   # 配置中心业务（含版本记录、回滚、导入导出）
│   │   ├── environment_service.go              # 环境 CRUD
│   │   ├── cluster_service.go                  # 集群状态/指标
│   │   ├── user_service.go                     # 用户 CRUD
│   │   └── audit_service.go                    # 审计日志查询
│   ├── middleware/
│   │   ├── jwt.go                              # JWT 认证中间件
│   │   ├── cors.go                             # CORS 中间件
│   │   └── role.go                             # 角色权限中间件
│   ├── handler/
│   │   ├── response.go                         # 统一响应格式 + 错误码
│   │   ├── auth.go                             # 认证路由处理
│   │   ├── kv.go                               # KV 管理路由处理
│   │   ├── config_center.go                    # 配置中心路由处理
│   │   ├── watch.go                            # SSE Watch 路由处理
│   │   ├── cluster.go                          # 集群信息路由处理
│   │   ├── user.go                             # 用户管理路由处理
│   │   ├── audit.go                            # 审计日志路由处理
│   │   └── router.go                           # 路由注册
│   └── seed/seed.go                            # 初始 admin 用户创建
```

---

## Chunk 1: 项目基础设施（配置、数据库、模型、仓储层）

### Task 1: 项目依赖初始化

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: 安装核心依赖**

```bash
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/sqlite
go get github.com/spf13/viper
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
go get go.etcd.io/etcd/client/v3
go get gopkg.in/yaml.v3
```

- [ ] **Step 2: 验证 go.mod**

Run: `cat go.mod`
Expected: 所有依赖已列出

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add core dependencies"
```

### Task 2: 应用配置

**Files:**
- Create: `configs/config.yaml`
- Create: `internal/config/config.go`

- [ ] **Step 1: 创建默认配置文件 configs/config.yaml**

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
  secret: "change-me-in-production"
  expire_hours: 24

log:
  level: "info"
```

- [ ] **Step 2: 创建 internal/config/config.go**

```go
package config

import "github.com/spf13/viper"

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Etcd     EtcdConfig     `mapstructure:"etcd"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type EtcdConfig struct {
	Endpoints []string  `mapstructure:"endpoints"`
	Username  string    `mapstructure:"username"`
	Password  string    `mapstructure:"password"`
	TLS       TLSConfig `mapstructure:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./internal/config/...`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add configs/config.yaml internal/config/
git commit -m "feat: add application config with Viper"
```

### Task 3: GORM 数据模型

**Files:**
- Create: `internal/model/user.go`
- Create: `internal/model/environment.go`
- Create: `internal/model/config_revision.go`
- Create: `internal/model/audit_log.go`

- [ ] **Step 1: 创建四个模型文件**

`internal/model/user.go`:
```go
package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:16;not null;default:viewer" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

`internal/model/environment.go`:
```go
package model

type Environment struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;size:64;not null" json:"name"`
	KeyPrefix   string `gorm:"size:255;not null" json:"key_prefix"`
	Description string `gorm:"size:255" json:"description"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
}
```

`internal/model/config_revision.go`:
```go
package model

import "time"

type ConfigRevision struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	EnvironmentID uint      `gorm:"index;not null" json:"environment_id"`
	Key           string    `gorm:"size:512;not null;index" json:"key"`
	Value         string    `gorm:"type:text" json:"value"`
	PrevValue     string    `gorm:"type:text" json:"prev_value"`
	EtcdRevision  int64     `json:"etcd_revision"`
	Action        string    `gorm:"size:16;not null" json:"action"`
	Operator      uint      `json:"operator"`
	Comment       string    `gorm:"size:512" json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}
```

`internal/model/audit_log.go`:
```go
package model

import "time"

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id"`
	Action       string    `gorm:"size:64;not null;index" json:"action"`
	ResourceType string    `gorm:"size:64;not null" json:"resource_type"`
	ResourceKey  string    `gorm:"size:512" json:"resource_key"`
	Detail       string    `gorm:"type:text" json:"detail"`
	IP           string    `gorm:"size:45" json:"ip"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/model/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/model/
git commit -m "feat: add GORM data models"
```

### Task 4: Repository 接口 + TransactionManager 接口

**Files:**
- Create: `internal/store/repository.go`
- Create: `internal/store/transaction.go`

- [ ] **Step 1: 创建 internal/store/transaction.go**

```go
package store

import "context"

type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

- [ ] **Step 2: 创建 internal/store/repository.go**

```go
package store

import (
	"context"
	"time"

	"github.com/dysodeng/config-center/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	List(ctx context.Context, page, pageSize int) ([]model.User, int64, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uint) error
}

type EnvironmentRepository interface {
	Create(ctx context.Context, env *model.Environment) error
	GetByID(ctx context.Context, id uint) (*model.Environment, error)
	GetByName(ctx context.Context, name string) (*model.Environment, error)
	List(ctx context.Context) ([]model.Environment, error)
	Update(ctx context.Context, env *model.Environment) error
	Delete(ctx context.Context, id uint) error
}

type ConfigRevisionRepository interface {
	Create(ctx context.Context, rev *model.ConfigRevision) error
	ListByKey(ctx context.Context, envID uint, key string, page, pageSize int) ([]model.ConfigRevision, int64, error)
	GetByID(ctx context.Context, id uint) (*model.ConfigRevision, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	List(ctx context.Context, filter AuditLogFilter, page, pageSize int) ([]model.AuditLog, int64, error)
}

type AuditLogFilter struct {
	UserID       *uint
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./internal/store/...`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add internal/store/repository.go internal/store/transaction.go
git commit -m "feat: define repository and transaction manager interfaces"
```

### Task 5: SQLite 实现（DB 初始化 + TransactionManager + 四个 Repository）

**Files:**
- Create: `internal/store/sqlite/db.go`
- Create: `internal/store/sqlite/transaction.go`
- Create: `internal/store/sqlite/user_repository.go`
- Create: `internal/store/sqlite/environment_repository.go`
- Create: `internal/store/sqlite/config_revision_repository.go`
- Create: `internal/store/sqlite/audit_log_repository.go`

- [ ] **Step 1: 创建 internal/store/sqlite/db.go**

```go
package sqlite

import (
	"os"
	"path/filepath"

	"github.com/dysodeng/config-center/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB(dbPath string) (*gorm.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Environment{},
		&model.ConfigRevision{},
		&model.AuditLog{},
	); err != nil {
		return nil, err
	}
	return db, nil
}
```

- [ ] **Step 2: 创建 internal/store/sqlite/transaction.go**

```go
package sqlite

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx := tm.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// GetDB 从 context 中获取事务 DB，如果没有事务则返回原始 DB
func GetDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db.WithContext(ctx)
}
```

- [ ] **Step 3: 创建四个 Repository 实现**

`internal/store/sqlite/user_repository.go`:
```go
package sqlite

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return GetDB(ctx, r.db).Create(user).Error
}
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	return &u, GetDB(ctx, r.db).First(&u, id).Error
}
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	return &u, GetDB(ctx, r.db).Where("username = ?", username).First(&u).Error
}
func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	db := GetDB(ctx, r.db).Model(&model.User{})
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id ASC").Find(&users).Error
	return users, total, err
}
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return GetDB(ctx, r.db).Save(user).Error
}
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	return GetDB(ctx, r.db).Delete(&model.User{}, id).Error
}
```

`internal/store/sqlite/environment_repository.go`:
```go
package sqlite

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"gorm.io/gorm"
)

type EnvironmentRepository struct{ db *gorm.DB }

func NewEnvironmentRepository(db *gorm.DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

func (r *EnvironmentRepository) Create(ctx context.Context, env *model.Environment) error {
	return GetDB(ctx, r.db).Create(env).Error
}
func (r *EnvironmentRepository) GetByID(ctx context.Context, id uint) (*model.Environment, error) {
	var e model.Environment
	return &e, GetDB(ctx, r.db).First(&e, id).Error
}
func (r *EnvironmentRepository) GetByName(ctx context.Context, name string) (*model.Environment, error) {
	var e model.Environment
	return &e, GetDB(ctx, r.db).Where("name = ?", name).First(&e).Error
}
func (r *EnvironmentRepository) List(ctx context.Context) ([]model.Environment, error) {
	var envs []model.Environment
	return envs, GetDB(ctx, r.db).Order("sort_order ASC, id ASC").Find(&envs).Error
}
func (r *EnvironmentRepository) Update(ctx context.Context, env *model.Environment) error {
	return GetDB(ctx, r.db).Save(env).Error
}
func (r *EnvironmentRepository) Delete(ctx context.Context, id uint) error {
	return GetDB(ctx, r.db).Delete(&model.Environment{}, id).Error
}
```

`internal/store/sqlite/config_revision_repository.go`:
```go
package sqlite

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"gorm.io/gorm"
)

type ConfigRevisionRepository struct{ db *gorm.DB }

func NewConfigRevisionRepository(db *gorm.DB) *ConfigRevisionRepository {
	return &ConfigRevisionRepository{db: db}
}

func (r *ConfigRevisionRepository) Create(ctx context.Context, rev *model.ConfigRevision) error {
	return GetDB(ctx, r.db).Create(rev).Error
}
func (r *ConfigRevisionRepository) ListByKey(ctx context.Context, envID uint, key string, page, pageSize int) ([]model.ConfigRevision, int64, error) {
	var revs []model.ConfigRevision
	var total int64
	db := GetDB(ctx, r.db).Model(&model.ConfigRevision{}).Where("environment_id = ? AND key = ?", envID, key)
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&revs).Error
	return revs, total, err
}
func (r *ConfigRevisionRepository) GetByID(ctx context.Context, id uint) (*model.ConfigRevision, error) {
	var rev model.ConfigRevision
	return &rev, GetDB(ctx, r.db).First(&rev, id).Error
}
```

`internal/store/sqlite/audit_log_repository.go`:
```go
package sqlite

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
	"gorm.io/gorm"
)

type AuditLogRepository struct{ db *gorm.DB }

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return GetDB(ctx, r.db).Create(log).Error
}
func (r *AuditLogRepository) List(ctx context.Context, filter store.AuditLogFilter, page, pageSize int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64
	db := GetDB(ctx, r.db).Model(&model.AuditLog{})
	if filter.UserID != nil {
		db = db.Where("user_id = ?", *filter.UserID)
	}
	if filter.Action != "" {
		db = db.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		db = db.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.StartTime != nil {
		db = db.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		db = db.Where("created_at <= ?", *filter.EndTime)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&logs).Error
	return logs, total, err
}
```

- [ ] **Step 4: 验证编译**

Run: `go build ./internal/...`
Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add internal/store/sqlite/
git commit -m "feat: implement SQLite repositories and transaction manager"
```

---

## Chunk 2: etcd 客户端 + 中间件 + 统一响应

### Task 6: etcd 客户端封装

**Files:**
- Create: `internal/etcd/client.go`

- [ ] **Step 1: 创建 internal/etcd/client.go**

```go
package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/dysodeng/config-center/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Client struct {
	cli *clientv3.Client
}

func NewClient(cfg config.EtcdConfig) (*Client, error) {
	etcdCfg := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: 5 * time.Second,
		Username:    cfg.Username,
		Password:    cfg.Password,
	}
	if cfg.TLS.Enabled {
		tlsCfg, err := newTLSConfig(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("etcd tls config: %w", err)
		}
		etcdCfg.TLS = tlsCfg
	}
	cli, err := clientv3.New(etcdCfg)
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error { return c.cli.Close() }

func (c *Client) Get(ctx context.Context, key string) (*clientv3.GetResponse, error) {
	return c.cli.Get(ctx, key)
}

func (c *Client) GetWithPrefix(ctx context.Context, prefix string, limit int64) (*clientv3.GetResponse, error) {
	opts := []clientv3.OpOption{clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend)}
	if limit > 0 {
		opts = append(opts, clientv3.WithLimit(limit))
	}
	return c.cli.Get(ctx, prefix, opts...)
}

func (c *Client) Put(ctx context.Context, key, value string) (*clientv3.PutResponse, error) {
	return c.cli.Put(ctx, key, value)
}

func (c *Client) Delete(ctx context.Context, key string) (*clientv3.DeleteResponse, error) {
	return c.cli.Delete(ctx, key)
}

func (c *Client) DeleteWithPrefix(ctx context.Context, prefix string) (*clientv3.DeleteResponse, error) {
	return c.cli.Delete(ctx, prefix, clientv3.WithPrefix())
}

func (c *Client) Watch(ctx context.Context, prefix string, rev int64) clientv3.WatchChan {
	opts := []clientv3.OpOption{clientv3.WithPrefix()}
	if rev > 0 {
		opts = append(opts, clientv3.WithRev(rev))
	}
	return c.cli.Watch(ctx, prefix, opts...)
}

func (c *Client) MemberList(ctx context.Context) (*clientv3.MemberListResponse, error) {
	return c.cli.MemberList(ctx)
}

func (c *Client) Status(ctx context.Context, endpoint string) (*clientv3.StatusResponse, error) {
	return c.cli.Status(ctx, endpoint)
}

func (c *Client) Endpoints() []string { return c.cli.Endpoints() }

func newTLSConfig(cfg config.TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/etcd/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/etcd/
git commit -m "feat: add etcd client wrapper"
```

### Task 7: 统一响应格式 + 错误码

**Files:**
- Create: `internal/handler/response.go`

- [ ] **Step 1: 创建 internal/handler/response.go**

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码
const (
	CodeSuccess         = 0
	CodeParamInvalid    = 10001
	CodeUnauthorized    = 10002
	CodeForbidden       = 10003
	CodeKeyExists       = 20001
	CodeKeyNotFound     = 20002
	CodeRevisionNotFound = 20003
	CodeEnvExists       = 20004
	CodeEnvHasConfigs   = 20005
	CodeEtcdConnFailed  = 30001
	CodeEtcdOpFailed    = 30002
	CodeUserExists      = 40001
	CodeAuthFailed      = 40002
	CodeImportFormat    = 50001
	CodeImportPartial   = 50002
	CodeInternalError   = 99999
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: CodeSuccess, Message: "ok", Data: data})
}

func OKPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "ok",
		Data:    PageData{List: list, Total: total, Page: page, PageSize: pageSize},
	})
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{Code: code, Message: message, Data: nil})
}

func FailUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{Code: CodeUnauthorized, Message: message, Data: nil})
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/response.go
git commit -m "feat: add unified response format and error codes"
```

### Task 8: JWT 中间件

**Files:**
- Create: `internal/middleware/jwt.go`

- [ ] **Step 1: 创建 internal/middleware/jwt.go**

```go
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/dysodeng/config-center/internal/handler"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			handler.FailUnauthorized(c, "missing token")
			c.Abort()
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			handler.FailUnauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	// Header: Authorization: Bearer <token>
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Query: ?token=xxx (for SSE)
	return c.Query("token")
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/middleware/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/jwt.go
git commit -m "feat: add JWT authentication middleware"
```

### Task 9: CORS 中间件

**Files:**
- Create: `internal/middleware/cors.go`

- [ ] **Step 1: 创建 internal/middleware/cors.go**

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/middleware/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/cors.go
git commit -m "feat: add CORS middleware"
```

### Task 10: 角色权限中间件

**Files:**
- Create: `internal/middleware/role.go`

- [ ] **Step 1: 创建 internal/middleware/role.go**

```go
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/handler"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			handler.Fail(c, handler.CodeForbidden, "admin role required")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireAdminForWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" {
			c.Next()
			return
		}
		role, _ := c.Get("role")
		if role != "admin" {
			handler.Fail(c, handler.CodeForbidden, "admin role required for write operations")
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/middleware/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/role.go
git commit -m "feat: add role-based access control middleware"
```

---

## Chunk 3: Service 层

### Task 11: Auth Service

**Files:**
- Create: `internal/service/auth_service.go`

- [ ] **Step 1: 创建 internal/service/auth_service.go**

```go
package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/dysodeng/config-center/internal/middleware"
	"github.com/dysodeng/config-center/internal/store"
)

type AuthService struct {
	userRepo  store.UserRepository
	jwtSecret string
	expireH   int
}

func NewAuthService(userRepo store.UserRepository, jwtSecret string, expireH int) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret, expireH: expireH}
}

type LoginResult struct {
	Token    string `json:"token"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid username or password")
	}
	claims := &middleware.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.expireH) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		Token:    tokenStr,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uint, oldPwd, newPwd string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPwd)); err != nil {
		return errors.New("old password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	return s.userRepo.Update(ctx, user)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/service/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/auth_service.go
git commit -m "feat: add auth service with login and password change"
```

### Task 12: User Service

**Files:**
- Create: `internal/service/user_service.go`

- [ ] **Step 1: 创建 internal/service/user_service.go**

```go
package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
)

type UserService struct {
	userRepo store.UserRepository
}

func NewUserService(userRepo store.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) Create(ctx context.Context, username, password, role string) (*model.User, error) {
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return nil, errors.New("username already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]model.User, int64, error) {
	return s.userRepo.List(ctx, page, pageSize)
}

func (s *UserService) Update(ctx context.Context, id uint, role string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.Role = role
	return s.userRepo.Update(ctx, user)
}

func (s *UserService) Delete(ctx context.Context, id uint) error {
	return s.userRepo.Delete(ctx, id)
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/service/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/user_service.go
git commit -m "feat: add user service with CRUD operations"
```

### Task 13: Environment Service

**Files:**
- Create: `internal/service/environment_service.go`

- [ ] **Step 1: 创建 internal/service/environment_service.go**

```go
package service

import (
	"context"
	"errors"

	"github.com/dysodeng/config-center/internal/etcd"
	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
)

type EnvironmentService struct {
	envRepo    store.EnvironmentRepository
	etcdClient *etcd.Client
}

func NewEnvironmentService(envRepo store.EnvironmentRepository, etcdClient *etcd.Client) *EnvironmentService {
	return &EnvironmentService{envRepo: envRepo, etcdClient: etcdClient}
}

func (s *EnvironmentService) Create(ctx context.Context, name, keyPrefix, description string, sortOrder int) (*model.Environment, error) {
	if _, err := s.envRepo.GetByName(ctx, name); err == nil {
		return nil, errors.New("environment already exists")
	}
	env := &model.Environment{
		Name:        name,
		KeyPrefix:   keyPrefix,
		Description: description,
		SortOrder:   sortOrder,
	}
	if err := s.envRepo.Create(ctx, env); err != nil {
		return nil, err
	}
	return env, nil
}

func (s *EnvironmentService) List(ctx context.Context) ([]model.Environment, error) {
	return s.envRepo.List(ctx)
}

func (s *EnvironmentService) Update(ctx context.Context, id uint, name, keyPrefix, description string, sortOrder int) error {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	env.Name = name
	env.KeyPrefix = keyPrefix
	env.Description = description
	env.SortOrder = sortOrder
	return s.envRepo.Update(ctx, env)
}

func (s *EnvironmentService) Delete(ctx context.Context, id uint) error {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// 检查环境下是否有配置
	resp, err := s.etcdClient.GetWithPrefix(ctx, env.KeyPrefix, 1)
	if err != nil {
		return err
	}
	if len(resp.Kvs) > 0 {
		return errors.New("environment has configs, cannot delete")
	}
	return s.envRepo.Delete(ctx, id)
}

func (s *EnvironmentService) GetByID(ctx context.Context, id uint) (*model.Environment, error) {
	return s.envRepo.GetByID(ctx, id)
}

func (s *EnvironmentService) GetByName(ctx context.Context, name string) (*model.Environment, error) {
	return s.envRepo.GetByName(ctx, name)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/service/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/environment_service.go
git commit -m "feat: add environment service"
```

### Task 14: Audit Service

**Files:**
- Create: `internal/service/audit_service.go`

- [ ] **Step 1: 创建 internal/service/audit_service.go**

```go
package service

import (
	"context"

	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
)

type AuditService struct {
	auditRepo store.AuditLogRepository
}

func NewAuditService(auditRepo store.AuditLogRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

func (s *AuditService) Log(ctx context.Context, userID uint, action, resourceType, resourceKey, detail, ip string) {
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceKey:  resourceKey,
		Detail:       detail,
		IP:           ip,
	})
}

func (s *AuditService) List(ctx context.Context, filter store.AuditLogFilter, page, pageSize int) ([]model.AuditLog, int64, error) {
	return s.auditRepo.List(ctx, filter, page, pageSize)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/service/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/audit_service.go
git commit -m "feat: add audit service"
```

### Task 15: KV Service

**Files:**
- Create: `internal/service/kv_service.go`

- [ ] **Step 1: 创建 internal/service/kv_service.go**

```go
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
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/service/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/kv_service.go
git commit -m "feat: add KV service"
```

### Task 16: Config Service（配置中心核心业务）

**Files:**
- Create: `internal/service/config_service.go`

- [ ] **Step 1: 创建 internal/service/config_service.go**

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dysodeng/config-center/internal/etcd"
	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
)

type ConfigService struct {
	etcdClient  *etcd.Client
	envRepo     store.EnvironmentRepository
	revisionRepo store.ConfigRevisionRepository
	txManager   store.TransactionManager
}

func NewConfigService(
	etcdClient *etcd.Client,
	envRepo store.EnvironmentRepository,
	revisionRepo store.ConfigRevisionRepository,
	txManager store.TransactionManager,
) *ConfigService {
	return &ConfigService{
		etcdClient:   etcdClient,
		envRepo:      envRepo,
		revisionRepo: revisionRepo,
		txManager:    txManager,
	}
}

type ConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *ConfigService) List(ctx context.Context, envName, prefix string) ([]ConfigItem, error) {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", envName)
	}
	fullPrefix := env.KeyPrefix + prefix
	resp, err := s.etcdClient.GetWithPrefix(ctx, fullPrefix, 0)
	if err != nil {
		return nil, err
	}
	items := make([]ConfigItem, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		shortKey := strings.TrimPrefix(string(kv.Key), env.KeyPrefix)
		items = append(items, ConfigItem{Key: shortKey, Value: string(kv.Value)})
	}
	return items, nil
}

func (s *ConfigService) Create(ctx context.Context, envName, key, value, comment string, operatorID uint) error {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}
	fullKey := env.KeyPrefix + key
	// 检查 key 是否已存在
	existing, err := s.etcdClient.Get(ctx, fullKey)
	if err != nil {
		return err
	}
	if len(existing.Kvs) > 0 {
		return errors.New("key already exists")
	}
	resp, err := s.etcdClient.Put(ctx, fullKey, value)
	if err != nil {
		return err
	}
	return s.revisionRepo.Create(ctx, &model.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		Value:         value,
		EtcdRevision:  resp.Header.Revision,
		Action:        "create",
		Operator:      operatorID,
		Comment:       comment,
	})
}

func (s *ConfigService) Update(ctx context.Context, envName, key, value, comment string, operatorID uint) error {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}
	fullKey := env.KeyPrefix + key
	existing, err := s.etcdClient.Get(ctx, fullKey)
	if err != nil {
		return err
	}
	var prevValue string
	if len(existing.Kvs) > 0 {
		prevValue = string(existing.Kvs[0].Value)
	}
	resp, err := s.etcdClient.Put(ctx, fullKey, value)
	if err != nil {
		return err
	}
	return s.revisionRepo.Create(ctx, &model.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		Value:         value,
		PrevValue:     prevValue,
		EtcdRevision:  resp.Header.Revision,
		Action:        "update",
		Operator:      operatorID,
		Comment:       comment,
	})
}

func (s *ConfigService) Delete(ctx context.Context, envName, key string, operatorID uint) error {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return fmt.Errorf("environment not found: %s", envName)
	}
	fullKey := env.KeyPrefix + key
	existing, err := s.etcdClient.Get(ctx, fullKey)
	if err != nil {
		return err
	}
	var prevValue string
	if len(existing.Kvs) > 0 {
		prevValue = string(existing.Kvs[0].Value)
	}
	resp, err := s.etcdClient.Delete(ctx, fullKey)
	if err != nil {
		return err
	}
	return s.revisionRepo.Create(ctx, &model.ConfigRevision{
		EnvironmentID: env.ID,
		Key:           key,
		PrevValue:     prevValue,
		EtcdRevision:  resp.Header.Revision,
		Action:        "delete",
		Operator:      operatorID,
	})
}

func (s *ConfigService) Revisions(ctx context.Context, envName, key string, page, pageSize int) ([]model.ConfigRevision, int64, error) {
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return nil, 0, fmt.Errorf("environment not found: %s", envName)
	}
	return s.revisionRepo.ListByKey(ctx, env.ID, key, page, pageSize)
}

func (s *ConfigService) Rollback(ctx context.Context, envName, key string, revisionID, operatorID uint) error {
	rev, err := s.revisionRepo.GetByID(ctx, revisionID)
	if err != nil {
		return errors.New("revision not found")
	}
	return s.Update(ctx, envName, key, rev.Value, fmt.Sprintf("rollback to revision %d", revisionID), operatorID)
}

func (s *ConfigService) Export(ctx context.Context, envName, format string) ([]byte, error) {
	items, err := s.List(ctx, envName, "")
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(items))
	for _, item := range items {
		m[item.Key] = item.Value
	}
	switch format {
	case "json":
		return json.MarshalIndent(m, "", "  ")
	case "yaml":
		return yaml.Marshal(m)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

type ImportResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  []string `json:"failed,omitempty"`
}

func (s *ConfigService) Import(ctx context.Context, envName string, data []byte, dryRun bool, operatorID uint) (*ImportResult, error) {
	var configs map[string]string
	// 尝试 JSON 解析，失败则尝试 YAML
	if err := json.Unmarshal(data, &configs); err != nil {
		if err := yaml.Unmarshal(data, &configs); err != nil {
			return nil, errors.New("invalid import format, expected JSON or YAML")
		}
	}
	result := &ImportResult{Total: len(configs)}
	if dryRun {
		result.Success = result.Total
		return result, nil
	}
	env, err := s.envRepo.GetByName(ctx, envName)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", envName)
	}
	for key, value := range configs {
		fullKey := env.KeyPrefix + key
		existing, _ := s.etcdClient.Get(ctx, fullKey)
		action := "create"
		var prevValue string
		if len(existing.Kvs) > 0 {
			action = "update"
			prevValue = string(existing.Kvs[0].Value)
		}
		resp, err := s.etcdClient.Put(ctx, fullKey, value)
		if err != nil {
			result.Failed = append(result.Failed, key)
			continue
		}
		_ = s.revisionRepo.Create(ctx, &model.ConfigRevision{
			EnvironmentID: env.ID,
			Key:           key,
			Value:         value,
			PrevValue:     prevValue,
			EtcdRevision:  resp.Header.Revision,
			Action:        action,
			Operator:      operatorID,
			Comment:       "import",
		})
		result.Success++
	}
	return result, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/service/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/config_service.go
git commit -m "feat: add config service with versioning, rollback, import/export"
```

### Task 17: Cluster Service

**Files:**
- Create: `internal/service/cluster_service.go`

- [ ] **Step 1: 创建 internal/service/cluster_service.go**

```go
package service

import (
	"context"

	"github.com/dysodeng/config-center/internal/etcd"
)

type ClusterService struct {
	etcdClient *etcd.Client
}

func NewClusterService(etcdClient *etcd.Client) *ClusterService {
	return &ClusterService{etcdClient: etcdClient}
}

type MemberInfo struct {
	ID         uint64   `json:"id"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peer_urls"`
	ClientURLs []string `json:"client_urls"`
	IsLearner  bool     `json:"is_learner"`
}

type ClusterStatus struct {
	Members []MemberInfo `json:"members"`
	Leader  uint64       `json:"leader"`
}

func (s *ClusterService) Status(ctx context.Context) (*ClusterStatus, error) {
	resp, err := s.etcdClient.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	status := &ClusterStatus{}
	for _, m := range resp.Members {
		status.Members = append(status.Members, MemberInfo{
			ID:         m.ID,
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
			ClientURLs: m.ClientURLs,
			IsLearner:  m.IsLearner,
		})
	}
	// 获取 leader 信息
	endpoints := s.etcdClient.Endpoints()
	if len(endpoints) > 0 {
		sr, err := s.etcdClient.Status(ctx, endpoints[0])
		if err == nil {
			status.Leader = sr.Leader
		}
	}
	return status, nil
}

type ClusterMetrics struct {
	Version    string            `json:"version"`
	DBSize     int64             `json:"db_size"`
	LeaderID   uint64            `json:"leader_id"`
	MemberCount int              `json:"member_count"`
	Health     map[string]bool   `json:"health"`
}

func (s *ClusterService) Metrics(ctx context.Context) (*ClusterMetrics, error) {
	endpoints := s.etcdClient.Endpoints()
	metrics := &ClusterMetrics{
		Health: make(map[string]bool),
	}
	for _, ep := range endpoints {
		sr, err := s.etcdClient.Status(ctx, ep)
		if err != nil {
			metrics.Health[ep] = false
			continue
		}
		metrics.Health[ep] = true
		if metrics.Version == "" {
			metrics.Version = sr.Version
			metrics.DBSize = sr.DbSize
			metrics.LeaderID = sr.Leader
		}
	}
	resp, err := s.etcdClient.MemberList(ctx)
	if err == nil {
		metrics.MemberCount = len(resp.Members)
	}
	return metrics, nil
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/service/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/service/cluster_service.go
git commit -m "feat: add cluster service for status and metrics"
```

---

## Chunk 4: Handler 层 + 路由注册

### Task 18: Auth Handler

**Files:**
- Create: `internal/handler/auth.go`

- [ ] **Step 1: 创建 internal/handler/auth.go**

```go
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
	userSvc *service.UserService
}

func NewAuthHandler(authSvc *service.AuthService, userSvc *service.UserService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userSvc: userSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "username and password required")
		return
	}
	result, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		Fail(c, CodeAuthFailed, err.Error())
		return
	}
	OK(c, result)
}

func (h *AuthHandler) Profile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.userSvc.GetByID(c.Request.Context(), userID.(uint))
	if err != nil {
		Fail(c, CodeUnauthorized, "user not found")
		return
	}
	OK(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	OK(c, nil)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "old_password and new_password required")
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.authSvc.ChangePassword(c.Request.Context(), userID.(uint), req.OldPassword, req.NewPassword); err != nil {
		Fail(c, CodeAuthFailed, err.Error())
		return
	}
	OK(c, nil)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/auth.go
git commit -m "feat: add auth handler"
```

### Task 19: KV Handler

**Files:**
- Create: `internal/handler/kv.go`

- [ ] **Step 1: 创建 internal/handler/kv.go**

```go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/service"
)

type KVHandler struct {
	kvSvc    *service.KVService
	auditSvc *service.AuditService
}

func NewKVHandler(kvSvc *service.KVService, auditSvc *service.AuditService) *KVHandler {
	return &KVHandler{kvSvc: kvSvc, auditSvc: auditSvc}
}

func (h *KVHandler) Get(c *gin.Context) {
	key := c.Query("key")
	prefix := c.Query("prefix")

	if key != "" {
		item, err := h.kvSvc.Get(c.Request.Context(), key)
		if err != nil {
			Fail(c, CodeEtcdOpFailed, err.Error())
			return
		}
		if item == nil {
			Fail(c, CodeKeyNotFound, "key not found")
			return
		}
		OK(c, item)
		return
	}

	if prefix == "" {
		prefix = "/"
	}
	limit := int64(50)
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	items, err := h.kvSvc.List(c.Request.Context(), prefix, limit)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	OK(c, items)
}

func (h *KVHandler) Create(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "key is required")
		return
	}
	// 检查是否已存在
	existing, _ := h.kvSvc.Get(c.Request.Context(), req.Key)
	if existing != nil {
		Fail(c, CodeKeyExists, "key already exists")
		return
	}
	if err := h.kvSvc.Put(c.Request.Context(), req.Key, req.Value); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "create", "kv", req.Key, "", c.ClientIP())
	OK(c, nil)
}

func (h *KVHandler) Update(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "key is required")
		return
	}
	if err := h.kvSvc.Put(c.Request.Context(), req.Key, req.Value); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "update", "kv", req.Key, "", c.ClientIP())
	OK(c, nil)
}

func (h *KVHandler) Delete(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		Fail(c, CodeParamInvalid, "key is required")
		return
	}
	if err := h.kvSvc.Delete(c.Request.Context(), key); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "delete", "kv", key, "", c.ClientIP())
	OK(c, nil)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/kv.go
git commit -m "feat: add KV handler"
```

### Task 20: Config Center Handler

**Files:**
- Create: `internal/handler/config_center.go`

- [ ] **Step 1: 创建 internal/handler/config_center.go**

```go
package handler

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/service"
)

type ConfigCenterHandler struct {
	configSvc *service.ConfigService
	envSvc    *service.EnvironmentService
	auditSvc  *service.AuditService
}

func NewConfigCenterHandler(
	configSvc *service.ConfigService,
	envSvc *service.EnvironmentService,
	auditSvc *service.AuditService,
) *ConfigCenterHandler {
	return &ConfigCenterHandler{configSvc: configSvc, envSvc: envSvc, auditSvc: auditSvc}
}

// --- 环境管理 ---

func (h *ConfigCenterHandler) ListEnvironments(c *gin.Context) {
	envs, err := h.envSvc.List(c.Request.Context())
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	OK(c, envs)
}

func (h *ConfigCenterHandler) CreateEnvironment(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		KeyPrefix   string `json:"key_prefix" binding:"required"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	env, err := h.envSvc.Create(c.Request.Context(), req.Name, req.KeyPrefix, req.Description, req.SortOrder)
	if err != nil {
		Fail(c, CodeEnvExists, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "create", "environment", req.Name, "", c.ClientIP())
	OK(c, env)
}

func (h *ConfigCenterHandler) UpdateEnvironment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, CodeParamInvalid, "invalid id")
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		KeyPrefix   string `json:"key_prefix" binding:"required"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	if err := h.envSvc.Update(c.Request.Context(), uint(id), req.Name, req.KeyPrefix, req.Description, req.SortOrder); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, nil)
}

func (h *ConfigCenterHandler) DeleteEnvironment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, CodeParamInvalid, "invalid id")
		return
	}
	if err := h.envSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		if err.Error() == "environment has configs, cannot delete" {
			Fail(c, CodeEnvHasConfigs, err.Error())
		} else {
			Fail(c, CodeEtcdOpFailed, err.Error())
		}
		return
	}
	OK(c, nil)
}

// --- 配置管理 ---

func (h *ConfigCenterHandler) ListConfigs(c *gin.Context) {
	env := c.Query("env")
	prefix := c.Query("prefix")
	if env == "" {
		Fail(c, CodeParamInvalid, "env is required")
		return
	}
	items, err := h.configSvc.List(c.Request.Context(), env, prefix)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	OK(c, items)
}

func (h *ConfigCenterHandler) CreateConfig(c *gin.Context) {
	var req struct {
		Env     string `json:"env" binding:"required"`
		Key     string `json:"key" binding:"required"`
		Value   string `json:"value"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.configSvc.Create(c.Request.Context(), req.Env, req.Key, req.Value, req.Comment, userID.(uint)); err != nil {
		if err.Error() == "key already exists" {
			Fail(c, CodeKeyExists, err.Error())
		} else {
			Fail(c, CodeEtcdOpFailed, err.Error())
		}
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "create", "config", req.Key, req.Env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Env     string `json:"env" binding:"required"`
		Key     string `json:"key" binding:"required"`
		Value   string `json:"value"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.configSvc.Update(c.Request.Context(), req.Env, req.Key, req.Value, req.Comment, userID.(uint)); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "update", "config", req.Key, req.Env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) DeleteConfig(c *gin.Context) {
	env := c.Query("env")
	key := c.Query("key")
	if env == "" || key == "" {
		Fail(c, CodeParamInvalid, "env and key are required")
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.configSvc.Delete(c.Request.Context(), env, key, userID.(uint)); err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "delete", "config", key, env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) Revisions(c *gin.Context) {
	env := c.Query("env")
	key := c.Query("key")
	if env == "" || key == "" {
		Fail(c, CodeParamInvalid, "env and key are required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	revs, total, err := h.configSvc.Revisions(c.Request.Context(), env, key, page, pageSize)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	OKPage(c, revs, total, page, pageSize)
}

func (h *ConfigCenterHandler) Rollback(c *gin.Context) {
	var req struct {
		Env        string `json:"env" binding:"required"`
		Key        string `json:"key" binding:"required"`
		RevisionID uint   `json:"revision_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.configSvc.Rollback(c.Request.Context(), req.Env, req.Key, req.RevisionID, userID.(uint)); err != nil {
		if err.Error() == "revision not found" {
			Fail(c, CodeRevisionNotFound, err.Error())
		} else {
			Fail(c, CodeEtcdOpFailed, err.Error())
		}
		return
	}
	h.auditSvc.Log(c.Request.Context(), userID.(uint), "rollback", "config", req.Key, req.Env, c.ClientIP())
	OK(c, nil)
}

func (h *ConfigCenterHandler) Export(c *gin.Context) {
	env := c.Query("env")
	format := c.DefaultQuery("format", "json")
	if env == "" {
		Fail(c, CodeParamInvalid, "env is required")
		return
	}
	data, err := h.configSvc.Export(c.Request.Context(), env, format)
	if err != nil {
		Fail(c, CodeEtcdOpFailed, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=config-"+env+"."+format)
	c.Data(200, "application/octet-stream", data)
}

func (h *ConfigCenterHandler) Import(c *gin.Context) {
	env := c.Query("env")
	dryRun := c.Query("dry_run") == "true"
	if env == "" {
		Fail(c, CodeParamInvalid, "env is required")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		Fail(c, CodeImportFormat, "failed to read request body")
		return
	}
	userID, _ := c.Get("user_id")
	result, err := h.configSvc.Import(c.Request.Context(), env, body, dryRun, userID.(uint))
	if err != nil {
		Fail(c, CodeImportFormat, err.Error())
		return
	}
	if len(result.Failed) > 0 {
		c.JSON(200, Response{Code: CodeImportPartial, Message: "partial import", Data: result})
		return
	}
	OK(c, result)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/config_center.go
git commit -m "feat: add config center handler with env and config CRUD"
```

### Task 21: Watch Handler（SSE）

**Files:**
- Create: `internal/handler/watch.go`

- [ ] **Step 1: 创建 internal/handler/watch.go**

```go
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/dysodeng/config-center/internal/etcd"
)

type WatchHandler struct {
	etcdClient *etcd.Client
}

func NewWatchHandler(etcdClient *etcd.Client) *WatchHandler {
	return &WatchHandler{etcdClient: etcdClient}
}

type WatchEvent struct {
	Type     string `json:"type"`
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	Revision int64  `json:"revision"`
}

func (h *WatchHandler) Watch(c *gin.Context) {
	prefix := c.Query("prefix")
	if prefix == "" {
		Fail(c, CodeParamInvalid, "prefix is required")
		return
	}

	var startRev int64
	if lastID := c.GetHeader("Last-Event-ID"); lastID != "" {
		if rev, err := strconv.ParseInt(lastID, 10, 64); err == nil {
			startRev = rev + 1
		}
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx := c.Request.Context()
	watchCh := h.etcdClient.Watch(ctx, prefix, startRev)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(30 * time.Minute)
	defer timeout.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		case <-timeout.C:
			return false
		case <-ticker.C:
			// 心跳
			fmt.Fprintf(w, ": heartbeat\n\n")
			return true
		case resp, ok := <-watchCh:
			if !ok {
				return false
			}
			if resp.CompactRevision > 0 {
				evt := WatchEvent{Type: "COMPACTED", Revision: resp.CompactRevision}
				data, _ := json.Marshal(evt)
				fmt.Fprintf(w, "event: kv_change\ndata: %s\nid: %d\n\n", data, resp.CompactRevision)
				return true
			}
			for _, ev := range resp.Events {
				evt := WatchEvent{
					Key:      string(ev.Kv.Key),
					Revision: ev.Kv.ModRevision,
				}
				if ev.Type == clientv3.EventTypePut {
					evt.Type = "PUT"
					evt.Value = string(ev.Kv.Value)
				} else {
					evt.Type = "DELETE"
				}
				data, _ := json.Marshal(evt)
				fmt.Fprintf(w, "event: kv_change\ndata: %s\nid: %d\n\n", data, ev.Kv.ModRevision)
			}
			return true
		}
	})
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/watch.go
git commit -m "feat: add SSE watch handler for real-time KV changes"
```

### Task 22: Cluster Handler

**Files:**
- Create: `internal/handler/cluster.go`

- [ ] **Step 1: 创建 internal/handler/cluster.go**

```go
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/service"
)

type ClusterHandler struct {
	clusterSvc *service.ClusterService
}

func NewClusterHandler(clusterSvc *service.ClusterService) *ClusterHandler {
	return &ClusterHandler{clusterSvc: clusterSvc}
}

func (h *ClusterHandler) Status(c *gin.Context) {
	status, err := h.clusterSvc.Status(c.Request.Context())
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, status)
}

func (h *ClusterHandler) Metrics(c *gin.Context) {
	metrics, err := h.clusterSvc.Metrics(c.Request.Context())
	if err != nil {
		Fail(c, CodeEtcdConnFailed, err.Error())
		return
	}
	OK(c, metrics)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/cluster.go
git commit -m "feat: add cluster handler"
```

### Task 23: User Handler

**Files:**
- Create: `internal/handler/user.go`

- [ ] **Step 1: 创建 internal/handler/user.go**

```go
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/service"
)

type UserHandler struct {
	userSvc  *service.UserService
	auditSvc *service.AuditService
}

func NewUserHandler(userSvc *service.UserService, auditSvc *service.AuditService) *UserHandler {
	return &UserHandler{userSvc: userSvc, auditSvc: auditSvc}
}

func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}
	users, total, err := h.userSvc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OKPage(c, users, total, page, pageSize)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	user, err := h.userSvc.Create(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		if err.Error() == "username already exists" {
			Fail(c, CodeUserExists, err.Error())
		} else {
			Fail(c, CodeInternalError, err.Error())
		}
		return
	}
	operatorID, _ := c.Get("user_id")
	h.auditSvc.Log(c.Request.Context(), operatorID.(uint), "create", "user", req.Username, "", c.ClientIP())
	OK(c, gin.H{"id": user.ID, "username": user.Username, "role": user.Role})
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, CodeParamInvalid, "invalid id")
		return
	}
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, err.Error())
		return
	}
	if err := h.userSvc.Update(c.Request.Context(), uint(id), req.Role); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		Fail(c, CodeParamInvalid, "invalid id")
		return
	}
	if err := h.userSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OK(c, nil)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/user.go
git commit -m "feat: add user handler"
```

### Task 24: Audit Handler

**Files:**
- Create: `internal/handler/audit.go`

- [ ] **Step 1: 创建 internal/handler/audit.go**

```go
package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/service"
	"github.com/dysodeng/config-center/internal/store"
)

type AuditHandler struct {
	auditSvc *service.AuditService
}

func NewAuditHandler(auditSvc *service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	filter := store.AuditLogFilter{
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
	}
	if uid := c.Query("user_id"); uid != "" {
		if id, err := strconv.ParseUint(uid, 10, 64); err == nil {
			u := uint(id)
			filter.UserID = &u
		}
	}
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			filter.StartTime = &t
		}
	}
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			filter.EndTime = &t
		}
	}

	logs, total, err := h.auditSvc.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		Fail(c, CodeInternalError, err.Error())
		return
	}
	OKPage(c, logs, total, page, pageSize)
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/audit.go
git commit -m "feat: add audit handler"
```

### Task 25: 路由注册

**Files:**
- Create: `internal/handler/router.go`

- [ ] **Step 1: 创建 internal/handler/router.go**

```go
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/middleware"
)

type Handlers struct {
	Auth         *AuthHandler
	KV           *KVHandler
	ConfigCenter *ConfigCenterHandler
	Watch        *WatchHandler
	Cluster      *ClusterHandler
	User         *UserHandler
	Audit        *AuditHandler
}

func RegisterRoutes(r *gin.Engine, h *Handlers, jwtSecret string) {
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")

	// 公开路由
	api.POST("/auth/login", h.Auth.Login)
	api.POST("/auth/logout", h.Auth.Logout)

	// 需要认证的路由
	auth := api.Group("", middleware.JWTAuth(jwtSecret))
	{
		// 认证相关
		auth.GET("/auth/profile", h.Auth.Profile)
		auth.PUT("/auth/password", h.Auth.ChangePassword)

		// KV 管理（viewer 只读，admin 可写）
		kv := auth.Group("/kv", middleware.RequireAdminForWrite())
		{
			kv.GET("", h.KV.Get)
			kv.POST("", h.KV.Create)
			kv.PUT("", h.KV.Update)
			kv.DELETE("", h.KV.Delete)
		}

		// 环境管理（admin only）
		envAdmin := auth.Group("/environments", middleware.RequireAdmin())
		{
			envAdmin.POST("", h.ConfigCenter.CreateEnvironment)
			envAdmin.PUT("/:id", h.ConfigCenter.UpdateEnvironment)
			envAdmin.DELETE("/:id", h.ConfigCenter.DeleteEnvironment)
		}
		auth.GET("/environments", h.ConfigCenter.ListEnvironments)

		// 配置中心（viewer 只读，admin 可写）
		configs := auth.Group("/configs", middleware.RequireAdminForWrite())
		{
			configs.GET("", h.ConfigCenter.ListConfigs)
			configs.POST("", h.ConfigCenter.CreateConfig)
			configs.PUT("", h.ConfigCenter.UpdateConfig)
			configs.DELETE("", h.ConfigCenter.DeleteConfig)
			configs.GET("/revisions", h.ConfigCenter.Revisions)
			configs.POST("/rollback", h.ConfigCenter.Rollback)
			configs.GET("/export", h.ConfigCenter.Export)
			configs.POST("/import", h.ConfigCenter.Import)
		}

		// Watch（SSE）
		auth.GET("/watch", h.Watch.Watch)

		// 集群信息
		auth.GET("/cluster/status", h.Cluster.Status)
		auth.GET("/cluster/metrics", h.Cluster.Metrics)

		// 用户管理（admin only）
		users := auth.Group("/users", middleware.RequireAdmin())
		{
			users.GET("", h.User.List)
			users.POST("", h.User.Create)
			users.PUT("/:id", h.User.Update)
			users.DELETE("/:id", h.User.Delete)
		}

		// 审计日志
		auth.GET("/audit-logs", h.Audit.List)
	}
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/handler/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/handler/router.go
git commit -m "feat: add route registration"
```

---

## Chunk 5: Seed + 入口 + 集成验证

### Task 26: 初始 admin 用户 Seed

**Files:**
- Create: `internal/seed/seed.go`

- [ ] **Step 1: 创建 internal/seed/seed.go**

```go
package seed

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/dysodeng/config-center/internal/model"
	"github.com/dysodeng/config-center/internal/store"
)

func CreateAdminUser(ctx context.Context, userRepo store.UserRepository) error {
	if _, err := userRepo.GetByUsername(ctx, "admin"); err == nil {
		return nil // admin 用户已存在
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := userRepo.Create(ctx, &model.User{
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         "admin",
	}); err != nil {
		return err
	}
	fmt.Println("========================================")
	fmt.Println("  Default admin user created:")
	fmt.Println("  Username: admin")
	fmt.Println("  Password: admin123")
	fmt.Println("  Please change the password after login!")
	fmt.Println("========================================")
	return nil
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./internal/seed/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add internal/seed/
git commit -m "feat: add admin user seed"
```

### Task 27: 服务入口 main.go

**Files:**
- Create: `cmd/server/main.go`

- [ ] **Step 1: 创建 cmd/server/main.go**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/config-center/internal/config"
	"github.com/dysodeng/config-center/internal/etcd"
	"github.com/dysodeng/config-center/internal/handler"
	"github.com/dysodeng/config-center/internal/seed"
	"github.com/dysodeng/config-center/internal/service"
	"github.com/dysodeng/config-center/internal/store/sqlite"
)

func main() {
	// 加载配置
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := sqlite.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	// 初始化 etcd 客户端
	etcdClient, err := etcd.NewClient(cfg.Etcd)
	if err != nil {
		log.Fatalf("failed to connect etcd: %v", err)
	}
	defer etcdClient.Close()

	// 初始化仓储层
	txManager := sqlite.NewTransactionManager(db)
	userRepo := sqlite.NewUserRepository(db)
	envRepo := sqlite.NewEnvironmentRepository(db)
	revisionRepo := sqlite.NewConfigRevisionRepository(db)
	auditRepo := sqlite.NewAuditLogRepository(db)

	// 创建初始 admin 用户
	if err := seed.CreateAdminUser(context.Background(), userRepo); err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	// 初始化 Service 层
	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	userSvc := service.NewUserService(userRepo)
	envSvc := service.NewEnvironmentService(envRepo, etcdClient)
	auditSvc := service.NewAuditService(auditRepo)
	kvSvc := service.NewKVService(etcdClient)
	configSvc := service.NewConfigService(etcdClient, envRepo, revisionRepo, txManager)
	clusterSvc := service.NewClusterService(etcdClient)

	// 初始化 Handler
	handlers := &handler.Handlers{
		Auth:         handler.NewAuthHandler(authSvc, userSvc),
		KV:           handler.NewKVHandler(kvSvc, auditSvc),
		ConfigCenter: handler.NewConfigCenterHandler(configSvc, envSvc, auditSvc),
		Watch:        handler.NewWatchHandler(etcdClient),
		Cluster:      handler.NewClusterHandler(clusterSvc),
		User:         handler.NewUserHandler(userSvc, auditSvc),
		Audit:        handler.NewAuditHandler(auditSvc),
	}

	// 启动 HTTP 服务
	r := gin.Default()
	handler.RegisterRoutes(r, handlers, cfg.JWT.Secret)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./cmd/server/...`
Expected: 无错误

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: add server entry point with dependency wiring"
```

### Task 28: 端到端编译验证

- [ ] **Step 1: 全量编译**

Run: `go build ./...`
Expected: 无错误

- [ ] **Step 2: 整理依赖**

Run: `go mod tidy`
Expected: go.mod 和 go.sum 更新

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: tidy go modules"
```

- [ ] **Step 4: 本地启动验证（可选，需要 etcd 实例）**

Run: `go run ./cmd/server/`
Expected: 看到 "Server starting on :8080" 和 admin 用户创建提示（如果是首次运行）

---

## 总结

| Chunk | Task 范围 | 描述 |
|-------|----------|------|
| 1 | Task 1-5 | 项目基础设施：依赖、配置、模型、仓储层 |
| 2 | Task 6-10 | etcd 客户端、响应格式、中间件（JWT/CORS/Role） |
| 3 | Task 11-17 | Service 层：Auth、User、Environment、Audit、KV、Config、Cluster |
| 4 | Task 18-25 | Handler 层 + 路由注册 |
| 5 | Task 26-28 | Seed、入口 main.go、集成验证 |