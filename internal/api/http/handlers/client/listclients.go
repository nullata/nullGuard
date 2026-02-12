// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"log"
	"net/http"
	"strconv"

	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	clientservice "nullguard/internal/service/client"
	serverservice "nullguard/internal/service/server"

	"github.com/gorilla/mux"
)

// ListClients returns a basic list of clients for a server
func ListClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	vars := mux.Vars(r)
	serverID, err := strconv.Atoi(vars["serverId"])
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid server ID", nil)
		return
	}

	server, err := serverservice.GetServerByID(serverID)
	if err != nil || server.ID == 0 {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Server not found", nil)
		return
	}

	clients, err := clientservice.GetServerClientsList(server)
	if err != nil {
		log.Printf("Error fetching clients: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Error fetching clients", nil)
		return
	}

	result := make([]models.ClientBasic, 0, len(clients))
	for _, c := range clients {
		result = append(result, models.ClientBasic{ID: c.ID, Name: c.Name})
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Client list", result)
}
