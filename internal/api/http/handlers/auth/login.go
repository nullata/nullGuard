// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"nullguard/internal/api/http/middleware"
	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	"nullguard/internal/service/auth"
)

// LoginHandler authenticates a user and creates a session
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	admin, err := auth.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusUnauthorized, constants.StatusError, err.Error(), nil)
		return
	}

	if err := middleware.CreateAuthSession(w, r, admin); err != nil {
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to create session", nil)
		return
	}

	log.Printf("User logged in: %s", req.Username)
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Login successful", nil)
}
