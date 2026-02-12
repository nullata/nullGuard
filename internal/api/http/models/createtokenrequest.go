// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

type CreateTokenRequest struct {
	Name          string `json:"name"`
	ExpiresInDays int    `json:"expires_in_days"` // 0 means never expires
}
