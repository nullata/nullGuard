// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

type SetupRequest struct {
	Username        string  `json:"username"`
	Password        string  `json:"password"`
	PasswordConfirm string  `json:"password_confirm"`
	PasswordHint    *string `json:"password_hint"`
}
