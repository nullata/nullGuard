// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nullguard/internal/api/http"
	"nullguard/internal/domain"
	"nullguard/internal/repository"
	"nullguard/internal/service/auth"
	"nullguard/internal/testutil"
	"nullguard/internal/testutil/httpsess"
)

// The auth middleware is the only gate in front of the whole API. This
// smoke test builds the REAL router and proves no protected route is
// reachable without credentials — a route registered on the base router by
// mistake (or a middleware-attach regression) would pass every handler
// unit test and the always-authenticated curl integration suite.

func seedAdmin(t *testing.T) {
	t.Helper()
	// direct insert: auth.CreateAdminAccount costs ~0.3s of bcrypt per call
	// and the middleware only needs the row to exist for RequireSetup
	if err := repository.CreateAdmin(&domain.Admin{
		Username:     "smoke-admin",
		PasswordHash: "not-a-real-hash",
	}); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
}

func TestRouter_ProtectedRoutesRejectUnauthenticated(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)
	seedAdmin(t) // past RequireSetup, so RequireAuthFlexible is what answers

	r := router.SetupRouter()

	cases := []struct {
		method string
		target string
	}{
		{http.MethodGet, "/api/v1/list-servers"},
		{http.MethodPost, "/api/v1/create-server"},
		{http.MethodPost, "/api/v1/deploy-server"},
		{http.MethodPut, "/api/v1/update-server"},
		{http.MethodDelete, "/api/v1/delete-server"},
		{http.MethodPost, "/api/v1/load-clients"},
		{http.MethodDelete, "/api/v1/delete-client"},
		{http.MethodPost, "/api/v1/tokens"},
		{http.MethodGet, "/api/v1/tokens"},
		{http.MethodPost, "/api/v1/logout"},
		{http.MethodPost, "/api/v1/change-password"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader("{}"))
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 without credentials (body: %.120s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRouter_PageRoutesRedirectUnauthenticated(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)
	seedAdmin(t)

	r := router.SetupRouter()

	for _, target := range []string{"/", "/server", "/client", "/admin/settings"} {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
				t.Fatalf("%s: status = %d, loc = %q, want 303 -> /login", target, rec.Code, rec.Header().Get("Location"))
			}
		})
	}
}

func TestRouter_HealthStaysPublic(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	r := router.SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200 without credentials", rec.Code)
	}
}

// A valid bearer token must pass the real middleware chain and reach a
// real handler (list-servers queries the test DB; no shell-outs).
func TestRouter_BearerReachesProtectedHandler(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)
	seedAdmin(t)

	plain, _, err := auth.GenerateApiToken(1, "smoke", 0, "")
	if err != nil {
		t.Fatalf("GenerateApiToken: %v", err)
	}

	r := router.SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-servers", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with valid bearer (body: %.200s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"success"`) {
		t.Fatalf("unexpected handler body: %.200s", rec.Body.String())
	}
}

// Once an admin exists, /api/v1/setup must refuse to create another one —
// at the HTTP layer, not just inside the service.
func TestRouter_SetupRefusedAfterAdminExists(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)
	seedAdmin(t)

	r := router.SetupRouter()

	body := `{"username":"mallory","password":"Str0ngPass","password_confirm":"Str0ngPass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatal("second admin account was accepted at /api/v1/setup")
	}
	var exists bool
	if _, err := repository.GetAdminByUsername("mallory"); err == nil {
		exists = true
	}
	if exists {
		t.Fatal("mallory was persisted despite refused setup")
	}
}

// AGENTS.md rule: /api/v1/tokens* are session-only even though the shared
// bearer middleware lets a valid token through - each handler re-reads the
// session store for admin_id and must 401 a bearer-only request. Pinning
// all three methods.
func TestRouter_TokensEndpointsRejectBearerAuth(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)
	seedAdmin(t)

	plain, _, err := auth.GenerateApiToken(1, "smoke", 0, "")
	if err != nil {
		t.Fatalf("GenerateApiToken: %v", err)
	}

	r := router.SetupRouter()
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/tokens"},
		{http.MethodPost, "/api/v1/tokens"},
		{http.MethodDelete, "/api/v1/tokens/1"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+plain)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s with bearer: status %d, want 401 (token self-management must be session-only)",
				tc.method, tc.path, rec.Code)
		}
	}
}
