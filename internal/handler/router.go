package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/middleware"
)

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
	Role         *RoleHandler
	Sync         *SyncHandler
}

func RegisterRoutes(r *gin.Engine, h *Handlers, jwtSecret string, roleRepo domain.RoleRepository) {
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

		// 角色管理（仅超级管理员）
		roles := auth.Group("/roles", middleware.RequireSuper())
		{
			roles.GET("", h.Role.List)
			roles.GET("/:id", h.Role.GetByID)
			roles.POST("", h.Role.Create)
			roles.PUT("/:id", h.Role.Update)
			roles.DELETE("/:id", h.Role.Delete)
		}

		// 超管转移（仅超级管理员）
		auth.PUT("/users/:id/transfer-super", middleware.RequireSuper(), h.User.TransferSuper)

		// 配置同步检测与恢复（仅超级管理员）
		sync := auth.Group("/sync", middleware.RequireSuper())
		{
			sync.GET("/check", h.Sync.Check)
			sync.POST("/restore", h.Sync.Restore)
		}

		// KV 管理
		kv := auth.Group("/kv", middleware.RequirePermission("kv", roleRepo), middleware.FilterEnvironments(roleRepo))
		{
			kv.GET("", h.KV.Get)
			kv.POST("", h.KV.Create)
			kv.PUT("", h.KV.Update)
			kv.DELETE("", h.KV.Delete)
		}

		// 环境列表（所有登录用户可访问，按角色过滤）
		auth.GET("/environments", middleware.FilterEnvironments(roleRepo), h.ConfigCenter.ListEnvironments)

		// 环境管理（需要 environments 写权限）
		envWrite := auth.Group("/environments", middleware.RequirePermission("environments", roleRepo))
		{
			envWrite.POST("", h.ConfigCenter.CreateEnvironment)
			envWrite.PUT("/:id", h.ConfigCenter.UpdateEnvironment)
			envWrite.DELETE("/:id", h.ConfigCenter.DeleteEnvironment)
		}

		// 配置中心
		configs := auth.Group("/configs", middleware.RequirePermission("config", roleRepo), middleware.FilterEnvironments(roleRepo))
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
		auth.GET("/watch", middleware.RequirePermission("kv", roleRepo), middleware.FilterEnvironments(roleRepo), h.Watch.Watch)

		// 集群信息
		cluster := auth.Group("/cluster", middleware.RequirePermission("cluster", roleRepo))
		{
			cluster.GET("/status", h.Cluster.Status)
			cluster.GET("/metrics", h.Cluster.Metrics)
			cluster.GET("/member-statuses", h.Cluster.MemberStatuses)
			cluster.GET("/alarms", h.Cluster.Alarms)
		}

		// 用户管理
		users := auth.Group("/users", middleware.RequirePermission("users", roleRepo))
		{
			users.GET("", h.User.List)
			users.POST("", h.User.Create)
			users.PUT("/:id", h.User.Update)
			users.DELETE("/:id", h.User.Delete)
		}

		// 审计日志
		auth.GET("/audit-logs", middleware.RequirePermission("audit_logs", roleRepo), h.Audit.List)

		// 网关服务管理
		gateway := auth.Group("/gateway", middleware.RequirePermission("gateway", roleRepo), middleware.FilterEnvironments(roleRepo))
		{
			gateway.GET("", h.Gateway.List)
			gateway.PUT("/status", h.Gateway.UpdateStatus)
		}

		// gRPC 服务管理
		grpc := auth.Group("/grpc", middleware.RequirePermission("grpc", roleRepo), middleware.FilterEnvironments(roleRepo))
		{
			grpc.GET("", h.Grpc.List)
			grpc.PUT("/status", h.Grpc.UpdateStatus)
		}
	}
}
