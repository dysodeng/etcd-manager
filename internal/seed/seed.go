package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/dysodeng/etcd-manager/internal/domain"
)

// CreateAdminUser 确保系统有超级管理员
// 1. 已有超管 → 跳过
// 2. 没有超管但有 admin 用户 → 将 admin 升级为超管
// 3. 都没有 → 创建默认 admin 超管账号
func CreateAdminUser(ctx context.Context, userRepo domain.UserRepository) error {
	// 已有超级管理员，跳过
	if _, err := userRepo.GetSuperAdmin(ctx); err == nil {
		return nil
	}

	// 检查是否已有 admin 用户（旧数据迁移场景）
	if existingAdmin, err := userRepo.GetByUsername(ctx, "admin"); err == nil {
		// 将已有的 admin 用户升级为超级管理员
		existingAdmin.IsSuper = true
		existingAdmin.RoleID = nil
		if err := userRepo.Update(ctx, existingAdmin); err != nil {
			return fmt.Errorf("failed to upgrade admin to super: %w", err)
		}
		fmt.Println("========================================")
		fmt.Println("  Existing admin user upgraded to super admin")
		fmt.Println("========================================")
		return nil
	}

	// 全新安装，创建默认超管
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := userRepo.Create(ctx, &domain.User{
		Username:     "admin",
		PasswordHash: string(hash),
		IsSuper:      true,
		RoleID:       nil,
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

// CreateDefaultRoles 创建默认角色（管理员 + 观察者）
func CreateDefaultRoles(ctx context.Context, roleRepo domain.RoleRepository, envRepo domain.EnvironmentRepository) error {
	allModules := []string{"kv", "config", "gateway", "grpc", "users", "environments", "audit_logs", "cluster"}

	// 获取所有环境ID
	envs, _ := envRepo.List(ctx)
	envIDs := make([]uuid.UUID, len(envs))
	for i := range envs {
		envIDs[i] = envs[i].ID
	}

	// 创建管理员角色
	if _, err := roleRepo.GetByName(ctx, "管理员"); err != nil {
		adminRole := &domain.Role{
			Name:        "管理员",
			Description: "所有模块读写权限",
		}
		if err := roleRepo.Create(ctx, adminRole); err != nil {
			return err
		}
		perms := make([]domain.RolePermission, len(allModules))
		for i, m := range allModules {
			perms[i] = domain.RolePermission{Module: m, CanRead: true, CanWrite: true}
		}
		_ = roleRepo.SetPermissions(ctx, adminRole.ID, perms)
		_ = roleRepo.SetEnvironments(ctx, adminRole.ID, envIDs)
	}

	// 创建观察者角色
	if _, err := roleRepo.GetByName(ctx, "观察者"); err != nil {
		viewerRole := &domain.Role{
			Name:        "观察者",
			Description: "所有模块只读权限",
		}
		if err := roleRepo.Create(ctx, viewerRole); err != nil {
			return err
		}
		perms := make([]domain.RolePermission, len(allModules))
		for i, m := range allModules {
			perms[i] = domain.RolePermission{Module: m, CanRead: true, CanWrite: false}
		}
		_ = roleRepo.SetPermissions(ctx, viewerRole.ID, perms)
		_ = roleRepo.SetEnvironments(ctx, viewerRole.ID, envIDs)
	}

	return nil
}

// MigrateOldRoles 迁移旧的 role 字段数据到新的 RBAC 模型
// 旧数据中 users 表有 role 列（"admin"/"viewer"），需要：
// 1. 非超管的 role="admin" 用户 → 分配"管理员"角色
// 2. role="viewer" 用户 → 分配"观察者"角色
// 此函数是幂等的，已有 role_id 的用户不会被重新分配
func MigrateOldRoles(db *gorm.DB, roleRepo domain.RoleRepository) error {
	// 检查旧 role 列是否存在
	if !db.Migrator().HasColumn("users", "role") {
		return nil
	}

	// 获取管理员角色ID
	adminRole, err := roleRepo.GetByName(context.Background(), "管理员")
	if err != nil {
		return nil // 默认角色还没创建，跳过
	}

	// 获取观察者角色ID
	viewerRole, err := roleRepo.GetByName(context.Background(), "观察者")
	if err != nil {
		return nil
	}

	// 将旧的 role="admin" 且非超管且没有 role_id 的用户分配"管理员"角色
	result := db.Exec(
		"UPDATE users SET role_id = ? WHERE role = ? AND is_super = ? AND role_id IS NULL",
		adminRole.ID.String(), "admin", false,
	)
	if result.RowsAffected > 0 {
		fmt.Printf("  Migrated %d old admin users to '管理员' role\n", result.RowsAffected)
	}

	// 将旧的 role="viewer" 且没有 role_id 的用户分配"观察者"角色
	result = db.Exec(
		"UPDATE users SET role_id = ? WHERE role = ? AND role_id IS NULL",
		viewerRole.ID.String(), "viewer",
	)
	if result.RowsAffected > 0 {
		fmt.Printf("  Migrated %d old viewer users to '观察者' role\n", result.RowsAffected)
	}

	return nil
}
