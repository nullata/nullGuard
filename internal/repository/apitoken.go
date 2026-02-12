// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package repository

import (
	"log"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
)

// CreateApiToken creates a new API token in the database
func CreateApiToken(token *domain.ApiToken) error {
	if err := database.DB.Create(token).Error; err != nil {
		log.Printf("Error creating API token: %v", err)
		return err
	}
	return nil
}

// GetApiTokenByHash retrieves a token by its hash
func GetApiTokenByHash(tokenHash string) (*domain.ApiToken, error) {
	var token domain.ApiToken
	if err := database.DB.Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

// GetApiTokenByID retrieves a token by its ID
func GetApiTokenByID(id uint) (*domain.ApiToken, error) {
	var token domain.ApiToken
	if err := database.DB.First(&token, id).Error; err != nil {
		log.Printf("Error getting API token by ID: %v", err)
		return nil, err
	}
	return &token, nil
}

// ListApiTokensByAdminID retrieves all tokens for a specific admin
func ListApiTokensByAdminID(adminID uint) ([]domain.ApiToken, error) {
	var tokens []domain.ApiToken
	if err := database.DB.Where("admin_id = ? AND revoked_at IS NULL", adminID).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		log.Printf("Error listing API tokens: %v", err)
		return nil, err
	}
	return tokens, nil
}

// UpdateApiToken updates an existing API token
func UpdateApiToken(token *domain.ApiToken) error {
	if err := database.DB.Save(token).Error; err != nil {
		log.Printf("Error updating API token: %v", err)
		return err
	}
	return nil
}

// RevokeApiToken soft-deletes a token by setting RevokedAt timestamp
func RevokeApiToken(id uint, adminID uint) error {
	now := time.Now()
	result := database.DB.Model(&domain.ApiToken{}).
		Where("id = ? AND admin_id = ?", id, adminID).
		Update("revoked_at", now)

	if result.Error != nil {
		log.Printf("Error revoking API token: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return database.DB.Error // token not found or doesnt belong to admin
	}

	return nil
}
