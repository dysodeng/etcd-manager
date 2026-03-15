package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/config"
	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/etcd"
	"github.com/dysodeng/etcd-manager/internal/handler"
	"github.com/dysodeng/etcd-manager/internal/seed"
	"github.com/dysodeng/etcd-manager/internal/service"
	"github.com/dysodeng/etcd-manager/internal/store/pgsql"
	"github.com/dysodeng/etcd-manager/internal/store/sqlite"
)

func main() {
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 设置北京时间
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatalf("failed to load timezone: %v", err)
	}
	time.Local = loc

	var (
		txManager    domain.TransactionManager
		userRepo     domain.UserRepository
		envRepo      domain.EnvironmentRepository
		revisionRepo domain.ConfigRevisionRepository
		auditRepo    domain.AuditLogRepository
	)

	switch cfg.Database.Driver {
	case "postgres":
		db, err := pgsql.NewDB(cfg.Database.DSN, loc)
		if err != nil {
			log.Fatalf("failed to init database: %v", err)
		}
		txManager = pgsql.NewTransactionManager(db)
		userRepo = pgsql.NewUserRepository(db)
		envRepo = pgsql.NewEnvironmentRepository(db)
		revisionRepo = pgsql.NewConfigRevisionRepository(db)
		auditRepo = pgsql.NewAuditLogRepository(db)
	default: // sqlite
		db, err := sqlite.NewDB(cfg.Database.Path, loc)
		if err != nil {
			log.Fatalf("failed to init database: %v", err)
		}
		txManager = sqlite.NewTransactionManager(db)
		userRepo = sqlite.NewUserRepository(db)
		envRepo = sqlite.NewEnvironmentRepository(db)
		revisionRepo = sqlite.NewConfigRevisionRepository(db)
		auditRepo = sqlite.NewAuditLogRepository(db)
	}

	etcdClient, err := etcd.NewClient(cfg.Etcd)
	if err != nil {
		log.Fatalf("failed to connect etcd: %v", err)
	}
	defer etcdClient.Close()

	if err = seed.CreateAdminUser(context.Background(), userRepo); err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	userSvc := service.NewUserService(userRepo)
	envSvc := service.NewEnvironmentService(envRepo, etcdClient)
	auditSvc := service.NewAuditService(auditRepo)
	kvSvc := service.NewKVService(etcdClient)
	configSvc := service.NewConfigService(etcdClient, envRepo, revisionRepo, txManager)
	clusterSvc := service.NewClusterService(etcdClient)
	gatewaySvc := service.NewGatewayService(etcdClient)
	grpcSvc := service.NewGrpcServiceManager(etcdClient)

	handlers := &handler.Handlers{
		Auth:         handler.NewAuthHandler(authSvc, userSvc),
		KV:           handler.NewKVHandler(kvSvc, auditSvc),
		ConfigCenter: handler.NewConfigCenterHandler(configSvc, envSvc, auditSvc),
		Watch:        handler.NewWatchHandler(etcdClient),
		Cluster:      handler.NewClusterHandler(clusterSvc),
		User:         handler.NewUserHandler(userSvc, auditSvc),
		Audit:        handler.NewAuditHandler(auditSvc, userSvc),
		Gateway:      handler.NewGatewayHandler(gatewaySvc, auditSvc),
		Grpc:         handler.NewGrpcHandler(grpcSvc, auditSvc),
	}

	r := gin.Default()
	_ = r.SetTrustedProxies([]string{"0.0.0.0/0"})
	r.RemoteIPHeaders = []string{"X-Real-IP", "X-Forwarded-For"}
	handler.RegisterRoutes(r, handlers, cfg.JWT.Secret)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err = r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
