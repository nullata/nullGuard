// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

import "encoding/json"

type RawServerData struct {
	ID               json.Number `json:"serverId"`
	InterfaceName    string      `json:"interfaceName"`
	Comment          string      `json:"comment"`
	Address          string      `json:"address"`
	Port             json.Number `json:"port"`
	PublicKey        string      `json:"publicKey"`
	PrivateKey       string      `json:"privateKey"`
	PostUp           string      `json:"postUp"`
	PostDown         string      `json:"postDown"`
	WANAddress       string      `json:"wanAddress"`
	SupernetCIDR     string      `json:"supernetCidr"`
	BridgeNetworks   string      `json:"bridgeNetworks"`
	DefaultKeepAlive json.Number `json:"defaultKeepAlive"`
	AutoRestart      bool        `json:"autoRestart"`
}
