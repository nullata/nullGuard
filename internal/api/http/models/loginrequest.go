// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
