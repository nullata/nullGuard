// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"nullguard/internal/api/http/middleware"
	"nullguard/internal/api/http/models"
	"nullguard/internal/domain"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	"nullguard/internal/service/auth"

	"github.com/gorilla/mux"
)

// CreateApiTokenHandler creates a new API token
func CreateApiTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// get admin id from session (bearer auth not allowed for creating tokens)
	sess, _ := middleware.Store.Get(r, "auth") // err safe to ignore: gorilla returns a valid empty session on decode failure
	adminID, ok := sess.Values["admin_id"].(uint)
	if !ok {
		httputil.SendJSONResponse(w, http.StatusUnauthorized, constants.StatusError, "Unauthorized", nil)
		return
	}

	// parse request
	var req models.CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}
	defer r.Body.Close()

	// validate input
	if req.Name == "" {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Token name is required", nil)
		return
	}

	if len(req.Name) > 100 {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Token name too long (max 100 characters)", nil)
		return
	}

	// get client ip
	clientIP := httputil.GetClientIP(r)

	// generate token
	tokenString, token, err := auth.GenerateApiToken(adminID, req.Name, req.ExpiresInDays, clientIP)
	if err != nil {
		log.Printf("Error generating API token: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to generate token", nil)
		return
	}

	// build response with the actual token (only shown once)
	response := buildTokenResponse(token)
	response.Token = &tokenString

	log.Printf("API token created: %s (id: %d)", token.Name, token.ID)
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "API token created successfully. Save this token now, it will not be shown again.", response)
}

// ListApiTokensHandler lists all API tokens for the authenticated admin
func ListApiTokensHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// get admin id from session
	sess, _ := middleware.Store.Get(r, "auth") // err safe to ignore: gorilla returns a valid empty session on decode failure
	adminID, ok := sess.Values["admin_id"].(uint)
	if !ok {
		httputil.SendJSONResponse(w, http.StatusUnauthorized, constants.StatusError, "Unauthorized", nil)
		return
	}

	// get tokens
	tokens, err := auth.ListApiTokens(adminID)
	if err != nil {
		log.Printf("Error listing API tokens: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to list tokens", nil)
		return
	}

	// build response (without actual token values)
	var responses []models.TokenResponse
	for _, token := range tokens {
		responses = append(responses, buildTokenResponse(&token))
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Tokens retrieved", responses)
}

// RevokeApiTokenHandler revokes an API token
func RevokeApiTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// get admin id from session
	sess, _ := middleware.Store.Get(r, "auth") // err safe to ignore: gorilla returns a valid empty session on decode failure
	adminID, ok := sess.Values["admin_id"].(uint)
	if !ok {
		httputil.SendJSONResponse(w, http.StatusUnauthorized, constants.StatusError, "Unauthorized", nil)
		return
	}

	// get token id from url
	vars := mux.Vars(r)
	tokenIDStr := vars["id"]
	tokenID, err := strconv.ParseUint(tokenIDStr, 10, 32)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid token ID", nil)
		return
	}

	// revoke token
	if err := auth.RevokeApiToken(uint(tokenID), adminID); err != nil {
		log.Printf("Error revoking API token: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to revoke token", nil)
		return
	}

	log.Printf("API token revoked: id %d", uint(tokenID))
	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Token revoked successfully", nil)
}

// helper functions

func buildTokenResponse(token *domain.ApiToken) models.TokenResponse {
	response := models.TokenResponse{
		ID:          token.ID,
		Name:        token.Name,
		CreatedAt:   token.CreatedAt.Format(time.RFC3339),
		CreatedByIP: token.CreatedByIP,
		LastUsedIP:  token.LastUsedIP,
		IsExpired:   !token.IsValid(),
	}

	if token.ExpiresAt != nil {
		expiresStr := token.ExpiresAt.Format(time.RFC3339)
		response.ExpiresAt = &expiresStr

		// calculate days until expiry
		daysUntil := int(time.Until(*token.ExpiresAt).Hours() / 24)
		if daysUntil >= 0 {
			response.DaysUntilExpiry = &daysUntil
		}
	}

	if token.LastUsedAt != nil {
		lastUsedStr := token.LastUsedAt.Format(time.RFC3339)
		response.LastUsedAt = &lastUsedStr
	}

	return response
}
