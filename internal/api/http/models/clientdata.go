// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

import "encoding/json"

type RawClientData struct {
	ID                 json.Number `json:"clientId"`
	Name               string      `json:"name"`
	PublicKey          string      `json:"publicKey"`
	PrivateKey         string      `json:"privateKey"`
	AddressCidr        string      `json:"address"`
	AllowedIps         string      `json:"allowedIps"`
	DnsServers         string      `json:"dnsServers"`
	FullTunnel         bool        `json:"fullTunnel"`
	Keepalive          json.Number `json:"keepalive"`
	ServerID           json.Number `json:"serverId"`
	ServerSupernetCidr string      `json:"serverSupernet"`
	IsConnected        bool        `json:"isConnected"`
	TransferRx         int64       `json:"transferRx"`
	TransferTx         int64       `json:"transferTx"`
}
