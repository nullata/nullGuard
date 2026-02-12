// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

import "nullguard/internal/pkg/constants"

type StandardResponse struct {
	Timestamp string           `json:"timestamp"`
	Status    constants.Status `json:"status"`
	Message   string           `json:"message"`
	Data      interface{}      `json:"data"`
}
