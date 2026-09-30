// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package repository

import (
	"log"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"

	"gorm.io/gorm"
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

// TouchApiTokenLastUsed records the last-used timestamp and IP for a token.
// It only ever updates those two columns, and only while the token is not
// revoked, so a concurrent revoke can never be clobbered by a write-back
// of a stale row (which a full-row Save could do).
func TouchApiTokenLastUsed(id uint, usedByIP string) error {
	if err := database.DB.Model(&domain.ApiToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		UpdateColumns(map[string]any{
			"last_used_at": time.Now(),
			"last_used_ip": usedByIP,
		}).Error; err != nil {
		log.Printf("Error updating API token last used: %v", err)
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
		return gorm.ErrRecordNotFound // token not found or doesn't belong to admin
	}

	return nil
}
