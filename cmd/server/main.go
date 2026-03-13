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
	cfgPath := "configs/config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := sqlite.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	etcdClient, err := etcd.NewClient(cfg.Etcd)
	if err != nil {
		log.Fatalf("failed to connect etcd: %v", err)
	}
	defer etcdClient.Close()

	txManager := sqlite.NewTransactionManager(db)
	userRepo := sqlite.NewUserRepository(db)
	envRepo := sqlite.NewEnvironmentRepository(db)
	revisionRepo := sqlite.NewConfigRevisionRepository(db)
	auditRepo := sqlite.NewAuditLogRepository(db)

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

	handlers := &handler.Handlers{
		Auth:         handler.NewAuthHandler(authSvc, userSvc),
		KV:           handler.NewKVHandler(kvSvc, auditSvc),
		ConfigCenter: handler.NewConfigCenterHandler(configSvc, envSvc, auditSvc),
		Watch:        handler.NewWatchHandler(etcdClient),
		Cluster:      handler.NewClusterHandler(clusterSvc),
		User:         handler.NewUserHandler(userSvc, auditSvc),
		Audit:        handler.NewAuditHandler(auditSvc),
	}

	r := gin.Default()
	handler.RegisterRoutes(r, handlers, cfg.JWT.Secret)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err = r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
