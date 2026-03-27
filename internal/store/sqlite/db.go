package sqlite

import (
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB(dbPath string, loc *time.Location) (*gorm.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().In(loc) },
	})
	if err != nil {
		return nil, err
	}
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
