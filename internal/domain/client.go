// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package domain

type Client struct {
	ID          uint    `gorm:"primaryKey"`
	Name        string  `gorm:"type:varchar(255);not null"`
	PublicKey   string  `gorm:"type:text;not null"`
	PrivateKey  string  `gorm:"type:text;not null"`
	AddressCidr string  `gorm:"type:varchar(100);not null;uniqueIndex:clients_unique_address"`
	AllowedIps  string  `gorm:"type:varchar(100);not null"`
	DnsServers  string  `gorm:"type:varchar(100);not null;default:''"`
	FullTunnel  bool    `gorm:"not null"`
	Keepalive   int     `gorm:"type:int;not null;default:30"`
	ServerID    uint    `gorm:"not null;index:clients_server_FK"` // foreign key reference to the `server` table
	ExposedLans *string `gorm:"type:text"`                        // CIDRs of LANs reachable through this client (server-side route)
}

func (Client) TableName() string {
	return "client"
}
