// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

type ChangePasswordRequest struct {
	OldPassword        string  `json:"old_password"`
	NewPassword        string  `json:"new_password"`
	NewPasswordConfirm string  `json:"new_password_confirm"`
	NewPasswordHint    *string `json:"new_password_hint"`
}
