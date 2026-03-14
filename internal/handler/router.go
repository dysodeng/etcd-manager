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
	Gateway      *GatewayHandler
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

		// 网关服务管理（viewer 只读，admin 可下线）
		gateway := auth.Group("/gateway", middleware.RequireAdminForWrite())
		{
			gateway.GET("", h.Gateway.List)
			gateway.PUT("/status", h.Gateway.UpdateStatus)
		}
	}
}