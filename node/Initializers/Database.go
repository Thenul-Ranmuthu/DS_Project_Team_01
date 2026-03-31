package initializers

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/DS_node/pkg/retry"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectToDB() {

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// 1. Initial connection without database to ensure it exists
	dsnRoot := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port)

	err := retry.Do(3, "db_create", func() error {
		dbRoot, gormErr := gorm.Open(mysql.Open(dsnRoot), &gorm.Config{})
		if gormErr != nil {
			return gormErr
		}
		sqlDB, _ := dbRoot.DB()
		defer sqlDB.Close()
		return dbRoot.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbname)).Error
	})

	if err != nil {
		slog.Warn("Could not ensure database exists, proceeding anyway", "error", err, "dbname", dbname)
	}

	// 2. Main connection to the specific database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)

	var db *gorm.DB
	err = retry.Do(5, "db_connect", func() error {
		var gormErr error
		db, gormErr = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if gormErr == nil {
			sqlDB, dbErr := db.DB()
			if dbErr != nil {
				return dbErr
			}
			return sqlDB.Ping()
		}
		return gormErr
	})

	if err != nil {
		slog.Error("Failed to connect to database after retries", "error", err)
		os.Exit(1)
	}

	DB = db
}
