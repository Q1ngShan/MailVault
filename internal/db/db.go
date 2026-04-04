package db

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mailvault/internal/store"
)

func GetDBPath() string {
	var dataDir string
	if runtime.GOOS == "windows" {
		dataDir = os.Getenv("LOCALAPPDATA")
		if dataDir == "" {
			dataDir, _ = os.UserHomeDir()
		}
	} else if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "Library", "Application Support")
	} else {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".config")
	}
	return filepath.Join(dataDir, "mailvault", "mailvault.db")
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
