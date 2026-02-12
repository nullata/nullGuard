// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"errors"
	"log"
	"nullguard/internal/domain"
	"nullguard/internal/pkg/validation"
	"nullguard/internal/repository"
)

func CreateAdminAccount(username, password, passwordConfirm string, passwordHint *string) error {
	if err := validation.ValidateUsername(username); err != nil {
		return err
	}

	if err := validation.ValidatePassword(password); err != nil {
		return err
	}

	if err := validation.ValidatePasswordConfirmation(password, passwordConfirm); err != nil {
		return err
	}

	if err := validation.ValidatePasswordHint(passwordHint); err != nil {
		return err
	}

	exists, err := repository.CheckAdminExists()
	if err != nil {
		return err
	}
	if exists {
		return errors.New("Admin account already exists")
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return errors.New("Failed to hash password")
	}

	admin := &domain.Admin{
		Username:     username,
		PasswordHash: hashedPassword,
		PasswordHint: passwordHint,
	}

	if err := repository.CreateAdmin(admin); err != nil {
		return err
	}

	return nil
}

func AuthenticateUser(username, password string) (*domain.Admin, error) {
	admin, err := repository.GetAdminByUsername(username)
	if err != nil {
		log.Printf("Failed login attempt: username not found - %s", username)
		return nil, errors.New("Invalid username or password")
	}

	if err := VerifyPassword(admin.PasswordHash, password); err != nil {
		log.Printf("Failed login attempt: invalid password - %s", username)
		return nil, errors.New("Invalid username or password")
	}

	return admin, nil
}

func ChangePassword(adminID uint, oldPassword, newPassword, newPasswordConfirm string, newPasswordHint *string) error {
	admin, err := repository.GetFirstAdmin()
	if err != nil {
		return errors.New("Admin not found")
	}

	if admin.ID != adminID {
		return errors.New("Unauthorized")
	}

	if err := VerifyPassword(admin.PasswordHash, oldPassword); err != nil {
		log.Printf("Failed password change attempt: invalid old password - %s", admin.Username)
		return errors.New("Invalid old password")
	}

	if err := validation.ValidatePassword(newPassword); err != nil {
		return err
	}

	if err := validation.ValidatePasswordConfirmation(newPassword, newPasswordConfirm); err != nil {
		return err
	}

	if err := validation.ValidatePasswordHint(newPasswordHint); err != nil {
		return err
	}

	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return errors.New("Failed to hash password")
	}

	admin.PasswordHash = hashedPassword
	admin.PasswordHint = newPasswordHint

	if err := repository.UpdateAdmin(admin); err != nil {
		return err
	}

	return nil
}

// CheckAdminExists checks if an admin account exists
func CheckAdminExists() (bool, error) {
	return repository.CheckAdminExists()
}

// GetAdminByUsername retrieves an admin by username
func GetAdminByUsername(username string) (*domain.Admin, error) {
	return repository.GetAdminByUsername(username)
}

// GetPasswordHint retrieves the password hint for a username
func GetPasswordHint(username string) (*string, error) {
	admin, err := repository.GetAdminByUsername(username)
	if err != nil {
		return nil, err
	}
	return admin.PasswordHint, nil
}
