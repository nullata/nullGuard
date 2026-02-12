// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package validation

import (
	"errors"
	"regexp"
	"strings"
)

// Package-level cached regex patterns for password validation
var (
	upperCaseRegex *regexp.Regexp
	lowerCaseRegex *regexp.Regexp
	digitRegex     *regexp.Regexp
)

func init() {
	upperCaseRegex = regexp.MustCompile(`[A-Z]`)
	lowerCaseRegex = regexp.MustCompile(`[a-z]`)
	digitRegex = regexp.MustCompile(`[0-9]`)
}

func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return errors.New("Username must be at least 3 characters long")
	}

	if !AllowedNameRegex.MatchString(username) {
		return errors.New("Username can only contain alphanumeric characters, dots, dashes, and underscores")
	}

	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("Password must be at least 8 characters long")
	}

	hasUpper := upperCaseRegex.MatchString(password)
	hasLower := lowerCaseRegex.MatchString(password)
	hasDigit := digitRegex.MatchString(password)

	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("Password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}

func ValidatePasswordConfirmation(password, passwordConfirm string) error {
	if password != passwordConfirm {
		return errors.New("Passwords do not match")
	}
	return nil
}

func ValidatePasswordHint(hint *string) error {
	if hint != nil && len(*hint) > 255 {
		return errors.New("Password hint must be 255 characters or less")
	}
	return nil
}
