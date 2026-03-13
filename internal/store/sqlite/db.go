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