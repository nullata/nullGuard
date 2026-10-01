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

func DeployServer(w http.ResponseWriter, r *http.Request) {
	server, ok := validateAndGetServer(w, r)
	if !ok {
		return
	}

	// generate config + start run as one atomic operation under the
	// per-interface lock (#14)
	if err := serverservice.DeployServerLocked(*server); err != nil {
		log.Printf("Error deploying server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, err.Error(), nil)
		return
	}

	responseMessage := fmt.Sprintf("Server deployed and started: %s", server.InterfaceName)
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, responseMessage, nil)
}
