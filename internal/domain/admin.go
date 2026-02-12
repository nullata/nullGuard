// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package domain

import (
	"time"

	"gorm.io/gorm"
)

type Admin struct {
	ID           uint    `gorm:"primaryKey"`
	Username     string  `gorm:"type:varchar(100);not null;uniqueIndex"`
	PasswordHash string  `gorm:"type:varchar(255);not null"`
	PasswordHint *string `gorm:"type:varchar(255)"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
