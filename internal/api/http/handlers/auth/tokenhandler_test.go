// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/testutil"
	"nullguard/internal/testutil/httpsess"
)

func insertToken(t *testing.T, adminID uint, hash string) *domain.ApiToken {
	t.Helper()
	tok := &domain.ApiToken{
		AdminID:   adminID,
		TokenHash: hash,
		Name:      "test",
		CreatedAt: time.Now(),
	}
	if err := database.DB.Create(tok).Error; err != nil {
		t.Fatalf("insert token: %v", err)
	}
	return tok
}

// Regression test for #13: the revoke endpoint must report failure (404)
// when the token id doesn't exist or belongs to another admin, instead of
// the old always-successful response.
func TestRevokeApiTokenHandler_UnknownTokenReturns404(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	req := httpsess.AuthedRequest(t, http.MethodDelete, "/api/v1/tokens/99999", 1,
		map[string]string{"id": "99999"})
	rec := httptest.NewRecorder()

	RevokeApiTokenHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestRevokeApiTokenHandler_ForeignTokenReturns404(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	foreign := insertToken(t, 2, "hash-foreign")

	req := httpsess.AuthedRequest(t, http.MethodDelete, "/api/v1/tokens/1", 1,
		map[string]string{"id": "1"})
	rec := httptest.NewRecorder()

	RevokeApiTokenHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for another admin's token (body: %s)", rec.Code, rec.Body.String())
	}

	// the foreign token must remain untouched
	var got domain.ApiToken
	if err := database.DB.First(&got, foreign.ID).Error; err != nil {
		t.Fatalf("reload foreign token: %v", err)
	}
	if got.RevokedAt != nil {
		t.Fatal("another admin's token was revoked")
	}
}

func TestRevokeApiTokenHandler_ValidTokenReturns200(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	tok := insertToken(t, 1, "hash-mine")

	req := httpsess.AuthedRequest(t, http.MethodDelete, "/api/v1/tokens/1", 1,
		map[string]string{"id": "1"})
	rec := httptest.NewRecorder()

	RevokeApiTokenHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var got domain.ApiToken
	if err := database.DB.First(&got, tok.ID).Error; err != nil {
		t.Fatalf("reload token: %v", err)
	}
	if got.RevokedAt == nil {
		t.Fatal("token not revoked after 200 response")
	}
}

func TestRevokeApiTokenHandler_InvalidIDReturns400(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	req := httpsess.AuthedRequest(t, http.MethodDelete, "/api/v1/tokens/abc", 1,
		map[string]string{"id": "abc"})
	rec := httptest.NewRecorder()

	RevokeApiTokenHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
}

// sanity check that the error body uses the standard envelope
func TestRevokeApiTokenHandler_NotFoundUsesStandardEnvelope(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	req := httpsess.AuthedRequest(t, http.MethodDelete, "/api/v1/tokens/4242", 1,
		map[string]string{"id": "4242"})
	rec := httptest.NewRecorder()

	RevokeApiTokenHandler(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v (%s)", err, rec.Body.String())
	}
	if body["status"] != "error" {
		t.Fatalf("status field = %v, want \"error\"", body["status"])
	}
	if body["message"] != "Token not found" {
		t.Fatalf("message field = %v, want \"Token not found\"", body["message"])
	}
}
