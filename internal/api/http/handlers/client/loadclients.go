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
	serverservice "nullguard/internal/service/server"
)

func LoadClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	var rawData models.RawServerData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in LoadClients: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	serverID, err := httputil.ConvertJsonNumToIntPtr(rawData.ID)
	if err != nil {
		log.Printf("Error converting raw data to server: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	server, err := serverservice.GetServerByID(*serverID)
	if err != nil {
		log.Printf("Error fetching server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Error fetching server", nil)
		return
	}

	clients, err := clientservice.GetServerClientsList(server)
	if err != nil {
		log.Printf("Error fetching clients: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Error fetching clients", nil)
		return
	}

	peerStatuses := serverservice.GetPeerStatuses(server.InterfaceName)
	clientData := clientservice.MapClientsToRawData(clients, peerStatuses)

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Success", clientData)
}
