// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the work factor for password hashing. It is a variable
// rather than a literal so tests can lower it (bcrypt is deliberately slow
// by design, and -race instrumentation amplifies that; the suite hashes
// many passwords). Production always runs it at 14.
var bcryptCost = 14

func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
