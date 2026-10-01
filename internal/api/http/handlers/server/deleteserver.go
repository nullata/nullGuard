// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"errors"
	"log"
	"net/http"

	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	serverservice "nullguard/internal/service/server"
)

func DeleteServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// decode json to struct
	var rawData models.DeployData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in DeleteServer: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Error parsing JSON", nil)
		return
	}

	serverID, err := serverservice.ValidateDeployData(rawData)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return
	}

	server, err := serverservice.GetServerByIDAndInterfaceName(*serverID, rawData.InterfaceName)
	if err != nil {
		log.Printf("Error fetching server by ID and interface name: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Could not find requested server", nil)
		return
	}

	// check + cascade client delete + row delete + conf removal run as one
	// locked critical section: the inactive check inside the lock closes
	// the TOCTOU where a concurrent deploy started the interface and this
	// handler then deleted the conf underneath it (#18)
	if err := serverservice.DeleteServerIfInactiveLocked(server); err != nil {
		if errors.Is(err, serverservice.ErrServerActive) {
			httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Server is active. Please stop the server before deleting", nil)
			return
		}
		log.Printf("Error deleting server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "There was a problem deleting requested server", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Server configuration deleted", nil)
}
