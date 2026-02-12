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

func UpdateClient(w http.ResponseWriter, r *http.Request) {
	rawData, ok := httputil.ValidateAndDecodeJSON[models.RawClientData](w, r, http.MethodPut)
	if !ok {
		return
	}

	newClient, err := clientservice.ConvertRawToClient(*rawData)
	if err != nil {
		log.Printf("Error converting raw data to client: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	if err := clientservice.Validate(&newClient); err != nil {
		log.Printf("Error during client field validation: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return
	}

	oldClient, err := clientservice.GetClientByIDAndServerID(newClient)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Could not find requested client", nil)
		return
	}

	if err := clientservice.UpdateClient(&oldClient, newClient); err != nil {
		log.Printf("Error updating client: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "There was a problem while updating the client configuration", nil)
		return
	}

	clientservice.ClearClientSessionData(w, r)

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Client updated. Changes will be applied after the server is restarted", nil)
}
