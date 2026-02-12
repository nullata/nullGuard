// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

type TokenResponse struct {
	ID              uint    `json:"id"`
	Name            string  `json:"name"`
	Token           *string `json:"token,omitempty"` // only included on creation
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	LastUsedAt      *string `json:"last_used_at,omitempty"`
	CreatedByIP     string  `json:"created_by_ip,omitempty"`
	LastUsedIP      string  `json:"last_used_ip,omitempty"`
	IsExpired       bool    `json:"is_expired"`
	DaysUntilExpiry *int    `json:"days_until_expiry,omitempty"`
}
