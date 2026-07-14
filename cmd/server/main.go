package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/dysodeng/etcd-manager/internal/config"
	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/etcd"
	"github.com/dysodeng/etcd-manager/internal/handler"
	"github.com/dysodeng/etcd-manager/internal/logging"
	"github.com/dysodeng/etcd-manager/internal/middleware"
	"github.com/dysodeng/etcd-manager/internal/seed"
	"github.com/dysodeng/etcd-manager/internal/service"
	"github.com/dysodeng/etcd-manager/internal/store/pgsql"
	"github.com/dysodeng/etcd-manager/internal/store/sqlite"
)

func main() {
	fallbackLogger, _ := logging.NewJSONLogger(os.Stdout, "info")
	slog.SetDefault(fallbackLogger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger, validLevel := logging.NewJSONLogger(os.Stdout, cfg.Log.Level)
	slog.SetDefault(logger)
	if !validLevel {
		logger.Warn("unknown log level; using info", "configured_level", cfg.Log.Level)
	}

	// 设置北京时间
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return fmt.Errorf("load timezone: %w", err)
	}
	time.Local = loc

	var (
		db           *gorm.DB
		txManager    domain.TransactionManager
		userRepo     domain.UserRepository
		envRepo      domain.EnvironmentRepository
		revisionRepo domain.ConfigRevisionRepository
		auditRepo    domain.AuditLogRepository
		roleRepo     domain.RoleRepository
	)

	switch cfg.Database.Driver {
	case "postgres":
		pgDB, err := pgsql.NewDB(cfg.Database.DSN, loc)
		if err != nil {
			return fmt.Errorf("initialize postgres database: %w", err)
		}
		db = pgDB
		txManager = pgsql.NewTransactionManager(db)
		userRepo = pgsql.NewUserRepository(db)
		envRepo = pgsql.NewEnvironmentRepository(db)
		revisionRepo = pgsql.NewConfigRevisionRepository(db)
		auditRepo = pgsql.NewAuditLogRepository(db)
		roleRepo = pgsql.NewRoleRepository(db)
	default: // sqlite
		sqliteDB, err := sqlite.NewDB(cfg.Database.Path, loc)
		if err != nil {
			return fmt.Errorf("initialize sqlite database: %w", err)
		}
		db = sqliteDB
		txManager = sqlite.NewTransactionManager(db)
		userRepo = sqlite.NewUserRepository(db)
		envRepo = sqlite.NewEnvironmentRepository(db)
		revisionRepo = sqlite.NewConfigRevisionRepository(db)
		auditRepo = sqlite.NewAuditLogRepository(db)
		roleRepo = sqlite.NewRoleRepository(db)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql database: %w", err)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			logger.Error("close database", "error", err)
		}
	}()

	etcdClient, err := etcd.NewClient(cfg.Etcd)
	if err != nil {
		return fmt.Errorf("connect etcd: %w", err)
	}
	defer func() {
		if err := etcdClient.Close(); err != nil {
			logger.Error("close etcd client", "error", err)
		}
	}()

	if err = seed.CreateAdminUser(ctx, userRepo); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	if err = seed.CreateDefaultRoles(ctx, roleRepo, envRepo); err != nil {
		return fmt.Errorf("seed default roles: %w", err)
	}
	// 迁移旧的 role 字段数据到新 RBAC 模型
	if err = seed.MigrateOldRoles(db, roleRepo); err != nil {
		return fmt.Errorf("migrate old roles: %w", err)
	}

	authSvc := service.NewAuthService(userRepo, roleRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	userSvc := service.NewUserService(userRepo, roleRepo, txManager)
	roleSvc := service.NewRoleService(roleRepo, userRepo)
	envSvc := service.NewEnvironmentService(envRepo, etcdClient)
	auditSvc := service.NewAuditService(auditRepo)
	kvSvc := service.NewKVService(etcdClient)
	configSvc := service.NewConfigService(etcdClient, envRepo, revisionRepo)
	clusterSvc := service.NewClusterService(etcdClient)
	gatewaySvc := service.NewGatewayService(etcdClient)
	grpcSvc := service.NewGrpcServiceManager(etcdClient)
	syncSvc := service.NewSyncService(etcdClient, envRepo, revisionRepo)

	handlers := &handler.Handlers{
		Auth:         handler.NewAuthHandler(authSvc),
		KV:           handler.NewKVHandler(kvSvc, auditSvc),
		ConfigCenter: handler.NewConfigCenterHandler(configSvc, envSvc, auditSvc),
		Watch:        handler.NewWatchHandler(etcdClient),
		Cluster:      handler.NewClusterHandler(clusterSvc),
		User:         handler.NewUserHandler(userSvc, auditSvc),
		Audit:        handler.NewAuditHandler(auditSvc, userSvc),
		Gateway:      handler.NewGatewayHandler(gatewaySvc, envSvc, auditSvc),
		Grpc:         handler.NewGrpcHandler(grpcSvc, envSvc, auditSvc),
		Role:         handler.NewRoleHandler(roleSvc, auditSvc),
		Sync:         handler.NewSyncHandler(syncSvc, auditSvc),
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.AccessLogger(logger), middleware.Recovery(logger))
	if err := r.SetTrustedProxies([]string{"0.0.0.0/0"}); err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}
	r.RemoteIPHeaders = []string{"X-Real-IP", "X-Forwarded-For"}
	handler.RegisterRoutes(r, handlers, cfg.JWT.Secret, userRepo, roleRepo)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	server := newHTTPServer(addr, r, cfg.Server)
	logger.Info("server starting", "address", addr)
	if err := serveHTTP(ctx, server, listener, cfg.Server.ShutdownTimeout); err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	logger.Info("server stopped")
	return nil
}
