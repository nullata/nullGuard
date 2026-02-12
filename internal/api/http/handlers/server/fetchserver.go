// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"log"
	"net/http"

	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	serverservice "nullguard/internal/service/server"
)

// FetchServer retrieves server details by ID
func FetchServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	var rawData models.RawServerData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in FetchServer: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Error parsing JSON", nil)
		return
	}

	serverID, err := httputil.ConvertJsonNumToIntPtr(rawData.ID)
	if err != nil {
		log.Printf("Error converting raw data to server: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	// fetch server data by ID
	server, err := serverservice.GetServerByID(*serverID)
	if err != nil {
		log.Printf("Error fetching server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Error fetching server", nil)
		return
	}

	isActive, _ := serverservice.IsServerActive(server)

	responseData := models.ServerWithStatus{
		Server:   server,
		IsActive: isActive,
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Server data", responseData)
}
