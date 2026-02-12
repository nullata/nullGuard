// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"fmt"
	"log"
	"net/http"

	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	serverservice "nullguard/internal/service/server"
)

func StopServer(w http.ResponseWriter, r *http.Request) {
	server, ok := validateAndGetServer(w, r)
	if !ok {
		return
	}

	isActive, err := serverservice.IsServerActive(*server)
	if err != nil {
		log.Printf("Error checking server status in StopServer: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Could not check server status", nil)
		return
	}

	if !isActive {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Server is not currently active", nil)
		return
	}

	if err := serverservice.StopServer(*server); err != nil {
		log.Printf("Error stopping server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, err.Error(), nil)
		return
	}

	responseMessage := fmt.Sprintf("Server stopped: %s", server.InterfaceName)
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, responseMessage, nil)
}
