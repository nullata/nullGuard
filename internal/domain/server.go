// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package domain

type Server struct {
	ID               uint    `gorm:"primaryKey"`
	InterfaceName    string  `gorm:"type:varchar(100);not null;uniqueIndex:server_unique_name"`
	Comment          string  `gorm:"type:varchar(255)"`
	Address          string  `gorm:"type:varchar(100);not null;uniqueIndex:server_unique_address"`
	Port             int     `gorm:"not null;uniqueIndex:server_unique_port"`
	PublicKey        string  `gorm:"type:text;not null"`
	PrivateKey       string  `gorm:"type:text;not null"`
	PostUp           *string `gorm:"type:text"`
	PostDown         *string `gorm:"type:text"`
	WANAddress       string  `gorm:"type:varchar(100);not null"`
	SupernetCidr     *string `gorm:"type:varchar(100)"`
	DefaultKeepalive *int    `gorm:"type:int"`
	AutoRestart      bool    `gorm:"default:false"`
}

func (Server) TableName() string {
	return "server"
}
