// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

type ServerBasic struct {
	ID            uint   `json:"id"`
	InterfaceName string `json:"interfaceName"`
	Address       string `json:"address"`
}
