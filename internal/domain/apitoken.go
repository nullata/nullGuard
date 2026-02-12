// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package domain

import "time"

// ApiToken represents an API authentication token
type ApiToken struct {
	ID          uint       `gorm:"primaryKey"`
	AdminID     uint       `gorm:"not null;index"`
	TokenHash   string     `gorm:"not null;unique;size:64"` // sha256 hash of the token
	Name        string     `gorm:"not null;size:100"`       // user-friendly name/description
	CreatedAt   time.Time  `gorm:"not null"`
	ExpiresAt   *time.Time `gorm:"index"` // null means never expires
	LastUsedAt  *time.Time // track when token was last used
	RevokedAt   *time.Time `gorm:"index"`   // soft delete - track when revoked
	CreatedByIP string     `gorm:"size:45"` // ip address that created the token
	LastUsedIP  string     `gorm:"size:45"` // ip address that last used the token
}

// IsValid checks if the token is still valid (not expired, not revoked)
func (t *ApiToken) IsValid() bool {
	if t.RevokedAt != nil {
		return false
	}

	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}
