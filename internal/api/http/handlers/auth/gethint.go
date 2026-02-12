// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"net/http"

	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	authservice "nullguard/internal/service/auth"
)

func GetPasswordHintHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Username is required", nil)
		return
	}

	hintPtr, err := authservice.GetPasswordHint(username)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusNotFound, constants.StatusError, "User not found", nil)
		return
	}

	hint := ""
	if hintPtr != nil {
		hint = *hintPtr
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Password hint retrieved", models.HintResponse{
		Hint: hint,
	})
}
