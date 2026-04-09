package db

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mailvault/internal/store"
)

func GetDBPath() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join("config", "mailvault.db")
	}
	return filepath.Join(filepath.Dir(exe), "config", "mailvault.db")
}

func Init() (*gorm.DB, error) {
	dbPath := GetDBPath()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&store.Account{}, &store.AccountType{}); err != nil {
		return nil, err
	}

	// Seed default account types if none exist
	var count int64
	db.Model(&store.AccountType{}).Count(&count)
	if count == 0 {
		defaults := []store.AccountType{
			{Code: "team", Label: "Team", Color: "#409EFF"},
			{Code: "member", Label: "Member", Color: "#67C23A"},
			{Code: "plus", Label: "Plus", Color: "#E6A23C"},
			{Code: "idle", Label: "Idle", Color: "#909399"},
		}
		db.Create(&defaults)
	}

	return db, nil
}
