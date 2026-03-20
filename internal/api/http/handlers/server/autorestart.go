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

func ToggleAutoRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	var rawData models.RawServerData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in ToggleAutoRestart: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	serverID, err := httputil.ConvertJsonNumToIntPtr(rawData.ID)
	if err != nil || serverID == nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "serverId is required", nil)
		return
	}

	if err := serverservice.SetAutoRestart(*serverID, rawData.AutoRestart); err != nil {
		log.Printf("Error setting auto-restart: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to update auto-restart setting", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Auto-restart updated", nil)
}
