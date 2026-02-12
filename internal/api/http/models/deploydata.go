// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package models

import "encoding/json"

type DeployData struct {
	ID            json.Number `json:"serverId"`
	InterfaceName string      `json:"interfaceName"`
}
