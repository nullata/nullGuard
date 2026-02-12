// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"log"
	"net/http"

	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	serverservice "nullguard/internal/service/server"
)

// ListServers returns a basic list of all WireGuard servers
func ListServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	servers, err := serverservice.GetServerList()
	if err != nil {
		log.Printf("Error fetching server list: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Error fetching server list", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Server list", servers)
}
