// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
