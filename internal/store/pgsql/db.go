package pgsql

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(dsn string, loc *time.Location) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().In(loc) },
	})
	if err != nil {
		return nil, err
	}
	// 确保 pg_uuidv7 扩展已启用
	db.Exec("CREATE EXTENSION IF NOT EXISTS pg_uuidv7")
	if err = db.AutoMigrate(
		&User{},
		&Role{},
		&RolePermission{},
		&RoleEnvironment{},
		&Environment{},
		&ConfigRevision{},
		&AuditLog{},
	); err != nil {
		return nil, err
	}
	return db, nil
}
