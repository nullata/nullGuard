// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/repository"
)

const (
	// TokenLength is the length of the generated token in bytes (before base64 encoding)
	TokenLength = 32
	// DefaultTokenExpiry is the default expiration time for tokens (0 = never expires)
	DefaultTokenExpiry = 0
)

// GenerateApiToken creates a new API token for the admin
func GenerateApiToken(adminID uint, name string, expiresInDays int, createdByIP string) (string, *domain.ApiToken, error) {
	// generate a secure random token
	tokenBytes := make([]byte, TokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode to base64 for the actual token string
	tokenString := base64.URLEncoding.EncodeToString(tokenBytes)

	// hash the token for storage (only store the hash)
	tokenHash := hashToken(tokenString)

	// calculate expiration
	var expiresAt *time.Time
	if expiresInDays > 0 {
		expiry := time.Now().AddDate(0, 0, expiresInDays)
		expiresAt = &expiry
	}

	// create the token record
	token := &domain.ApiToken{
		AdminID:     adminID,
		TokenHash:   tokenHash,
		Name:        name,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		CreatedByIP: createdByIP,
	}

	// save to database
	if err := repository.CreateApiToken(token); err != nil {
		return "", nil, fmt.Errorf("failed to save token: %w", err)
	}

	// return the plain token (this is the only time it will be visible)
	return tokenString, token, nil
}

// ValidateApiToken validates a bearer token and updates last used timestamp
func ValidateApiToken(tokenString string, usedByIP string) (*domain.ApiToken, error) {
	// hash the provided token
	tokenHash := hashToken(tokenString)

	// look up the token
	token, err := repository.GetApiTokenByHash(tokenHash)
	if err != nil {
		return nil, errors.New("invalid token")
	}

	// check if token is valid (not revoked, not expired)
	if !token.IsValid() {
		return nil, errors.New("token is revoked or expired")
	}

	// update last used timestamp and IP
	now := time.Now()
	token.LastUsedAt = &now
	token.LastUsedIP = usedByIP

	if err := repository.UpdateApiToken(token); err != nil {
		// log the error but dont fail the authentication
		// the token is still valid even if we cant update the last used time
		log.Printf("Warning: failed to update token last used time: %v", err)
	}

	return token, nil
}

// RevokeApiToken revokes a token by ID
func RevokeApiToken(tokenID uint, adminID uint) error {
	return repository.RevokeApiToken(tokenID, adminID)
}

// ListApiTokens returns all active tokens for an admin
func ListApiTokens(adminID uint) ([]domain.ApiToken, error) {
	return repository.ListApiTokensByAdminID(adminID)
}

// GetApiToken retrieves a specific token by ID
func GetApiToken(tokenID uint) (*domain.ApiToken, error) {
	return repository.GetApiTokenByID(tokenID)
}

// hashToken creates a sha256 hash of the token for secure storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
