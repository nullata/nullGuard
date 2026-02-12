// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"log"
	"net/http"

	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	clientservice "nullguard/internal/service/client"
)

// DeleteClient deletes a WireGuard client
func DeleteClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	var rawData models.RawClientData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in DeleteClient: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	client, err := clientservice.ConvertRawToClient(rawData)
	if err != nil {
		log.Printf("Error converting raw data to client: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	if err := clientservice.DeleteClientByIDAndServerID(client.ID, client.ServerID); err != nil {
		log.Printf("Error deleting client: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "There was a problem deleting the client", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Success", nil)
}
