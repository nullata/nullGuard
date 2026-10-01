// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/testutil"
)

// AGENTS.md rule: destructive server operations require BOTH serverId and
// a matching interfaceName. These tests pin the pairing behavior of the
// shared validator used by deploy/restart/stop handlers.
//
// Note: the no-serverId case is a known nil-deref (#21); its regression
// test lands with that fix and is deliberately not pinned here.

func seedServer(t *testing.T, name string) domain.Server {
	t.Helper()
	srv := domain.Server{
		InterfaceName: name,
		Address:       "10.8.0.1/24",
		Port:          51820,
		PublicKey:     "pubkey",
		PrivateKey:    "privkey",
		WANAddress:    "203.0.113.7",
	}
	if err := database.DB.Create(&srv).Error; err != nil {
		t.Fatalf("seed server: %v", err)
	}
	return srv
}

func callValidate(t *testing.T, method, body string) (*domain.Server, bool, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v1/deploy-server", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv, ok := validateAndGetServer(rec, req)
	return srv, ok, rec
}

func TestValidateAndGetServer_MethodGuard(t *testing.T) {
	testutil.NewTestDB(t)

	for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		srv, ok, rec := callValidate(t, m, `{"serverId":1,"interfaceName":"wg0"}`)
		if ok || srv != nil || rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: ok=%v status=%d, want blocked 405", m, ok, rec.Code)
		}
	}
}

func TestValidateAndGetServer_BadJSON(t *testing.T) {
	testutil.NewTestDB(t)

	srv, ok, rec := callValidate(t, http.MethodPost, `{not json`)
	if ok || srv != nil || rec.Code != http.StatusBadRequest {
		t.Fatalf("ok=%v status=%d, want blocked 400", ok, rec.Code)
	}
}

func TestValidateAndGetServer_EmptyInterfaceName(t *testing.T) {
	testutil.NewTestDB(t)
	seedServer(t, "wg0")

	srv, ok, rec := callValidate(t, http.MethodPost, `{"serverId":1,"interfaceName":"  "}`)
	if ok || srv != nil || rec.Code != http.StatusBadRequest {
		t.Fatalf("ok=%v status=%d, want blocked 400 (empty interface name)", ok, rec.Code)
	}
}

// The pairing rule: a valid id with a NON-matching interface name must not
// resolve to a server. (Current contract answers 500 "Could not find
// requested server"; pinned as-is — if this hardens to 400/404 the test
// updates deliberately.)
func TestValidateAndGetServer_MismatchedPairRejected(t *testing.T) {
	testutil.NewTestDB(t)
	seedServer(t, "wg0")

	srv, ok, rec := callValidate(t, http.MethodPost, `{"serverId":1,"interfaceName":"wg1"}`)
	if ok || srv != nil {
		t.Fatalf("mismatched interface name resolved server %+v", srv)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (current contract)", rec.Code)
	}
}

func TestValidateAndGetServer_MatchingPairSucceeds(t *testing.T) {
	testutil.NewTestDB(t)
	want := seedServer(t, "wg0")

	srv, ok, rec := callValidate(t, http.MethodPost, `{"serverId":1,"interfaceName":"wg0"}`)
	if !ok || srv == nil {
		t.Fatalf("valid pair blocked: ok=%v body=%s", ok, rec.Body.String())
	}
	if srv.ID != want.ID || srv.InterfaceName != "wg0" {
		t.Fatalf("resolved %+v, want id %d wg0", srv, want.ID)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("rec was written with %d on success path", rec.Code)
	}
}

// DeployData decodes serverId as json.Number to tolerate string-or-number
// clients; pin that both spellings resolve the same.
func TestValidateAndGetServer_IdNumberOrString(t *testing.T) {
	testutil.NewTestDB(t)
	seedServer(t, "wg0")

	for _, body := range []string{
		`{"serverId":1,"interfaceName":"wg0"}`,
		`{"serverId":"1","interfaceName":"wg0"}`,
	} {
		srv, ok, _ := callValidate(t, http.MethodPost, body)
		if !ok || srv == nil {
			t.Fatalf("body %s: valid numeric/string id rejected", body)
		}
	}

	// a non-numeric id string must be a clean 400, not a crash
	srv, ok, rec := callValidate(t, http.MethodPost, `{"serverId":"abc","interfaceName":"wg0"}`)
	if ok || srv != nil || rec.Code != http.StatusBadRequest {
		t.Fatalf("ok=%v status=%d, want blocked 400 for non-numeric id", ok, rec.Code)
	}
	var probe map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &probe); err != nil || probe["status"] != "error" {
		t.Fatalf("error envelope missing: %s", rec.Body.String())
	}
}
