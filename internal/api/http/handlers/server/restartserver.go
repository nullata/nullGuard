// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	serverservice "nullguard/internal/service/server"
)

func RestartServer(w http.ResponseWriter, r *http.Request) {
	server, ok := validateAndGetServer(w, r)
	if !ok {
		return
	}

	// the active check and the stop/regenerate/start sequence run under the
	// per-interface lock, so they can't interleave with auto-restarts, other
	// restarts, or deploys on the same interface (#14)
	if err := serverservice.RestartServerLocked(*server); err != nil {
		if errors.Is(err, serverservice.ErrServerNotActive) {
			httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Server is not currently active", nil)
			return
		}
		log.Printf("Error restarting server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, err.Error(), nil)
		return
	}

	responseMessage := fmt.Sprintf("Server restarted: %s", server.InterfaceName)
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, responseMessage, nil)
}
