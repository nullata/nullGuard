// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package domain

type Client struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"type:varchar(255);not null"`
	PublicKey   string `gorm:"type:text;not null"`
	PrivateKey  string `gorm:"type:text;not null"`
	AddressCidr string `gorm:"type:varchar(100);not null;uniqueIndex:clients_unique_address"`
	AllowedIps  string `gorm:"type:varchar(100);not null"`
	DnsServers  string `gorm:"type:varchar(100);not null;default:''"`
	FullTunnel  bool   `gorm:"type:tinyint(1);not null"`
	Keepalive   int    `gorm:"type:int;not null;default:30"`
	ServerID    uint   `gorm:"not null;index:clients_server_FK"` // foreign key reference to the `server` table
}

func (Client) TableName() string {
	return "client"
}
