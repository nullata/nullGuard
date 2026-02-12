// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"log"
	"net/http"
	"nullguard/internal/api/http/middleware"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
)

// LogoutHandler logs out the current user and destroys the session
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if err := middleware.DestroyAuthSession(w, r); err != nil {
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to logout", nil)
		return
	}

	log.Print("User logged out")
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Logout successful", nil)
}
