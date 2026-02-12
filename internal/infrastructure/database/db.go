// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package database

import (
	"fmt"
	"log"
	"time"

	"nullguard/internal/infrastructure/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"nullguard/internal/domain"
)

var DB *gorm.DB

// initialize the database connection
func InitDB() {
	user := config.GetEnv("DB_USER", "")
	password := config.GetEnv("DB_PASS", "")
	dbname := config.GetEnv("DB_NAME", "")
	host := config.GetEnv("DB_HOST", "")
	port := config.GetEnv("DB_PORT", "")

	if user == "" || password == "" ||
		dbname == "" || host == "" || port == "" {
		log.Fatal("failed to validate database configuration")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbname)
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	setConnectionPoolConf()

	migrateModels()

	log.Println("Database connection established")
}

func setConnectionPoolConf() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("failed to get raw database handle: %v", err)
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
	if newValue != nil && (*field == nil || *newValue != **field) {
		*field = newValue
	}
}
