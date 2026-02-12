// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	"nullguard/internal/service/auth"
)

func SetupHandler(w http.ResponseWriter, r *http.Request) {
	var req models.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	if err := auth.CreateAdminAccount(req.Username, req.Password, req.PasswordConfirm, req.PasswordHint); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return
	}

	log.Printf("Admin account created: %s", req.Username)
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Admin account created successfully", nil)
}
