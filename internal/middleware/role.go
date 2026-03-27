package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/domain"
	"github.com/dysodeng/etcd-manager/internal/response"
)

// RequireSuper 仅超级管理员可访问
func RequireSuper() gin.HandlerFunc {
	return func(c *gin.Context) {
		isSuper, _ := c.Get("is_super")
		if isSuper != true {
			response.Fail(c, response.CodeForbidden, "super admin required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequirePermission 检查角色的模块读/写权限
func RequirePermission(module string, roleRepo domain.RoleRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 超级管理员放行
		isSuper, _ := c.Get("is_super")
		if isSuper == true {
			c.Next()
			return
		}

		roleIDStr, _ := c.Get("role_id")
		rid, ok := roleIDStr.(string)
		if !ok || rid == "" {
			response.Fail(c, response.CodeForbidden, "no role assigned")
			c.Abort()
			return
		}

		roleID, err := uuid.Parse(rid)
		if err != nil {
			response.Fail(c, response.CodeForbidden, "invalid role")
			c.Abort()
			return
		}

		perms, err := roleRepo.GetPermissions(c.Request.Context(), roleID)
		if err != nil {
			response.Fail(c, response.CodeForbidden, "permission check failed")
			c.Abort()
			return
		}

		// 根据请求方法判断需要的权限
		action := "read"
		if c.Request.Method != "GET" {
			action = "write"
		}

		for _, p := range perms {
			if p.Module == module {
				if action == "read" && (p.CanRead || p.CanWrite) {
					c.Next()
					return
				}
				if action == "write" && p.CanWrite {
					c.Next()
					return
				}
			}
		}

		response.Fail(c, response.CodeForbidden, "permission denied")
		c.Abort()
	}
}

// FilterEnvironments 环境过滤中间件 - 将授权的环境ID列表放入 context
func FilterEnvironments(roleRepo domain.RoleRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		isSuper, _ := c.Get("is_super")
		if isSuper == true {
			// 超级管理员不做过滤
			c.Next()
			return
		}

		roleIDStr, _ := c.Get("role_id")
		rid, ok := roleIDStr.(string)
		if !ok || rid == "" {
			response.Fail(c, response.CodeForbidden, "no role assigned")
			c.Abort()
			return
		}

		roleID, err := uuid.Parse(rid)
		if err != nil {
			response.Fail(c, response.CodeForbidden, "invalid role")
			c.Abort()
			return
		}

		envIDs, err := roleRepo.GetEnvironmentIDs(c.Request.Context(), roleID)
		if err != nil {
			response.Fail(c, response.CodeForbidden, "environment check failed")
			c.Abort()
			return
		}

		c.Set("allowed_env_ids", envIDs)
		c.Next()
	}
}

// GetAllowedEnvIDs 从 context 获取授权的环境ID列表。返回 nil 表示超级管理员（不限制）
func GetAllowedEnvIDs(ctx context.Context) []uuid.UUID {
	if gc, ok := ctx.(*gin.Context); ok {
		if v, exists := gc.Get("allowed_env_ids"); exists {
			if ids, ok := v.([]uuid.UUID); ok {
				return ids
			}
		}
	}
	return nil
}
