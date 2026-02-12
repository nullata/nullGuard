// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package middleware

import (
	"context"
	"net/http"
	"strings"

	"nullguard/internal/pkg/httputil"
	"nullguard/internal/service/auth"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// AdminIDKey is the context key for storing the authenticated admin ID
	AdminIDKey contextKey = "admin_id"
	// AuthTypeKey is the context key for storing the authentication type (session or bearer)
	AuthTypeKey contextKey = "auth_type"
)

// RequireAuthFlexible allows both session-based auth and bearer token auth
func RequireAuthFlexible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip authentication for public endpoints
		if strings.HasPrefix(path, "/login") ||
			strings.HasPrefix(path, "/api/v1/login") ||
			strings.HasPrefix(path, "/setup") ||
			strings.HasPrefix(path, "/api/v1/setup") ||
			strings.HasPrefix(path, "/api/v1/get-password-hint") ||
			strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		// try bearer token authentication first
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// get client ip
			clientIP := httputil.GetClientIP(r)

			// validate the token
			apiToken, err := auth.ValidateApiToken(token, clientIP)
			if err != nil {
				http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// add admin id and auth type to context
			ctx := context.WithValue(r.Context(), AdminIDKey, apiToken.AdminID)
			ctx = context.WithValue(ctx, AuthTypeKey, "bearer")

			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// fall back to session authentication
		if !IsAuthenticated(r) {
			// for api endpoints, return json error
			if strings.HasPrefix(path, "/api/") {
				http.Error(w, "Unauthorized: Authentication required", http.StatusUnauthorized)
				return
			}
			// for page requests, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// session is valid, add admin id to context
		sess, _ := Store.Get(r, "auth") // err safe to ignore: gorilla returns a valid empty session on decode failure
		if adminID, ok := sess.Values["admin_id"].(uint); ok {
			ctx := context.WithValue(r.Context(), AdminIDKey, adminID)
			ctx = context.WithValue(ctx, AuthTypeKey, "session")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// if we get here, session was valid but no admin id found (shouldnt happen)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// GetAdminIDFromContext retrieves the admin ID from the request context
func GetAdminIDFromContext(r *http.Request) (uint, bool) {
	adminID, ok := r.Context().Value(AdminIDKey).(uint)
	return adminID, ok
}

// GetAuthTypeFromContext retrieves the authentication type from the request context
func GetAuthTypeFromContext(r *http.Request) (string, bool) {
	authType, ok := r.Context().Value(AuthTypeKey).(string)
	return authType, ok
}
