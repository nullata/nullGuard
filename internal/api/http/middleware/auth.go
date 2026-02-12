// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package middleware

import (
	"net/http"
	"nullguard/internal/repository"
	"strings"
)

// IsAuthenticated checks if the user is authenticated
func IsAuthenticated(r *http.Request) bool {
	sess, err := Store.Get(r, "auth")
	if err != nil {
		return false
	}

	authenticated, ok := sess.Values["authenticated"].(bool)
	if !ok || !authenticated {
		return false
	}

	return true
}

func RequireSetup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/setup") ||
			strings.HasPrefix(path, "/api/v1/setup") ||
			strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		adminExists, err := repository.CheckAdminExists()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !adminExists {
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/login") ||
			strings.HasPrefix(path, "/api/v1/login") ||
			strings.HasPrefix(path, "/setup") ||
			strings.HasPrefix(path, "/api/v1/setup") ||
			strings.HasPrefix(path, "/api/v1/get-password-hint") ||
			strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		if !IsAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ChainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
