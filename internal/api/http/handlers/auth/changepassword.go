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

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetAdminIDFromContext(r)
	if !ok {
		httputil.SendJSONResponse(w, http.StatusUnauthorized, constants.StatusError, "Not authenticated", nil)
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	if err := auth.ChangePassword(adminID, req.OldPassword, req.NewPassword, req.NewPasswordConfirm, req.NewPasswordHint); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return
	}

	log.Printf("Password changed successfully for admin ID: %d", adminID)
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Password changed successfully", nil)
}
