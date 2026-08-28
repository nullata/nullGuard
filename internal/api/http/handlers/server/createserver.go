// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"log"
	"net/http"
	"strings"

	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	serverservice "nullguard/internal/service/server"
	utils "nullguard/internal/service/wireguard"
)

func CreateServer(w http.ResponseWriter, r *http.Request) {
	rawData, ok := httputil.ValidateAndDecodeJSON[models.RawServerData](w, r, http.MethodPost)
	if !ok {
		return
	}

	// auto-generate keys if not provided
	if strings.TrimSpace(rawData.PublicKey) == "" || strings.TrimSpace(rawData.PrivateKey) == "" {
		privateKey, publicKey, err := utils.GenerateWgKeys("server")
		if err != nil {
			log.Printf("Error generating server keys: %v", err)
			httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to generate server keys", nil)
			return
		}
		rawData.PrivateKey = strings.TrimSpace(privateKey)
		rawData.PublicKey = strings.TrimSpace(publicKey)
	}

	// convert RawServerData to Server using the conversion function
	server, err := serverservice.ConvertRawToServer(*rawData)
	if err != nil {
		log.Printf("Error converting raw data to server: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	if err := serverservice.Validate(&server); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return
	}

	if err := serverservice.CreateServer(server); err != nil {
		if isDuplicateEntryError(err) {
			// duplicate unique-constraint violation (any database backend)
			httputil.SendJSONResponse(w, http.StatusConflict, constants.StatusError, "A server configuration using the same properties already exists", nil)
			return
		}
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "There was a problem creating an entry with the requested server configuration", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Server created", nil)
}
