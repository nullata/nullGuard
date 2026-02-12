// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

import "nullguard/internal/domain"

type ServerWithStatus struct {
	domain.Server
	IsActive bool `json:"IsActive"`
}
