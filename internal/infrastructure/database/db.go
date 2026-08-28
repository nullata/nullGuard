// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package database

import (
	"fmt"
	"log"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/config"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// initialize the database connection
func InitDB() {
	dbType := config.GetEnv("DB_TYPE", "mysql")

	if dbType == "sqlite" {
		dbUrl := config.GetEnv("DATABASE_URL", "nullguard.db")
		var err error
		DB, err = gorm.Open(sqlite.Open(dbUrl), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect sqlite database: %v", err)
		}
	} else if dbType == "postgres" {
		user := config.GetEnv("DB_USER", "")
		password := config.GetEnv("DB_PASS", "")
		dbname := config.GetEnv("DB_NAME", "")
		host := config.GetEnv("DB_HOST", "")
		port := config.GetEnv("DB_PORT", "")

		if user == "" || password == "" || dbname == "" || host == "" || port == "" {
			log.Fatal("failed to validate database configuration")
		}

		sslmode := config.GetEnv("DB_SSLMODE", "disable")
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC", host, user, password, dbname, port, sslmode)
		var err error
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect postgres database: %v", err)
		}
	} else {
		user := config.GetEnv("DB_USER", "")
		password := config.GetEnv("DB_PASS", "")
		dbname := config.GetEnv("DB_NAME", "")
		host := config.GetEnv("DB_HOST", "")
		port := config.GetEnv("DB_PORT", "")

		if user == "" || password == "" || dbname == "" || host == "" || port == "" {
			log.Fatal("failed to validate database configuration")
		}

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbname)
		var err error
		DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}
	}

	setConnectionPoolConf(dbType)

	migrateModels()

	log.Printf("Database connection established using %s", dbType)
}

func setConnectionPoolConf(dbType string) {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("failed to get raw database handle: %v", err)
	}

	if dbType == "sqlite" {
		// SQLite allows a single writer; one connection avoids "database is locked" errors
		sqlDB.SetMaxOpenConns(1)
		return
	}

	// connection pool conf
	sqlDB.SetMaxIdleConns(10)                  // idle connections in the pool
	sqlDB.SetMaxOpenConns(100)                 // open connections maximum number
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)  // connection idle timeout
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // maximum connection lifetime
}

func migrateModels() {
	// auto-migrate the schema based on domain models
	if err := DB.AutoMigrate(&domain.Server{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := DB.AutoMigrate(&domain.Client{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := DB.AutoMigrate(&domain.Admin{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	if err := DB.AutoMigrate(&domain.ApiToken{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
}

func UpdateFieldIfChanged[T comparable](field *T, newValue T) {
	if newValue != *field {
		*field = newValue
	}
}

func UpdatePointerFieldIfChanged[T comparable](field **T, newValue *T) {
	if newValue == nil && *field == nil {
		return
	}
	if newValue == nil || *field == nil || *newValue != **field {
		*field = newValue
	}
}
