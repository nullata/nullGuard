// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package repository

import (
	"log"
	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
)

func CheckAdminExists() (bool, error) {
	var count int64
	if err := database.DB.Model(&domain.Admin{}).Count(&count).Error; err != nil {
		log.Printf("Error checking if admin exists: %v", err)
		return false, err
	}
	return count > 0, nil
}

func CreateAdmin(admin *domain.Admin) error {
	if err := database.DB.Create(admin).Error; err != nil {
		log.Printf("Error creating admin: %v", err)
		return err
	}
	return nil
}

func GetAdminByUsername(username string) (*domain.Admin, error) {
	var admin domain.Admin
	if err := database.DB.Where("username = ?", username).First(&admin).Error; err != nil {
		log.Printf("Error getting admin by username: %v", err)
		return nil, err
	}
	return &admin, nil
}

func UpdateAdmin(admin *domain.Admin) error {
	if err := database.DB.Save(admin).Error; err != nil {
		log.Printf("Error updating admin: %v", err)
		return err
	}
	return nil
}

func GetFirstAdmin() (*domain.Admin, error) {
	var admin domain.Admin
	if err := database.DB.First(&admin).Error; err != nil {
		log.Printf("Error getting first admin: %v", err)
		return nil, err
	}
	return &admin, nil
}
