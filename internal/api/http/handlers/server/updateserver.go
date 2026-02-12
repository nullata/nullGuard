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

func UpdateServer(w http.ResponseWriter, r *http.Request) {
	rawData, ok := httputil.ValidateAndDecodeJSON[models.RawServerData](w, r, http.MethodPut)
	if !ok {
		return
	}

	newServer, err := serverservice.ConvertRawToServer(*rawData)
	if err != nil {
		log.Printf("Error converting raw data to server: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	if err := serverservice.Validate(&newServer); err != nil {
		log.Printf("Error during server field validation: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return
	}

	oldServer, err := serverservice.GetServerByIDAndKeys(int(newServer.ID), newServer.PublicKey, newServer.PrivateKey)
	if err != nil {
		log.Printf("Error fetching server by ID and keys: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Could not find requested server", nil)
		return
	}

	isServerActive, err := serverservice.IsServerActive(oldServer)
	if isServerActive {
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Server is active. Please stop the server before updating", nil)
		return
	} else if err != nil {
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, err.Error(), nil)
		return
	}

	if err := serverservice.UpdateServer(&oldServer, newServer); err != nil {
		log.Printf("Error updating server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "There was a problem while updating the server configuration", nil)
		return
	}

	// delete the old server configuration file if it exists
	if err := serverservice.DeleteServerConf(oldServer); err != nil {
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, err.Error(), nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Server updated", nil)
}
