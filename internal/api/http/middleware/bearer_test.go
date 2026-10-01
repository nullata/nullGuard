// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

// External test package: middleware must stay importable by
// testutil/httpsess, so these tests cannot live inside package middleware.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nullguard/internal/api/http/middleware"
	"nullguard/internal/service/auth"
	"nullguard/internal/testutil"
	"nullguard/internal/testutil/httpsess"
)

// okHandler is the sentinel "next" handler: reaching it means the
// middleware let the request through.
func okHandler() (http.Handler, *bool) {
	passed := new(bool)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*passed = true
		w.WriteHeader(http.StatusOK)
	}), passed
}

func TestRequireAuthFlexible_Unauthenticated(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	cases := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantPage   bool // redirect to /login instead of 401
	}{
		{"api path gets 401 json", http.MethodGet, "/api/v1/list-servers", http.StatusUnauthorized, false},
		{"api post gets 401", http.MethodPost, "/api/v1/deploy-server", http.StatusUnauthorized, false},
		{"page path redirects to login", http.MethodGet, "/server", http.StatusSeeOther, true},
		{"root redirects to login", http.MethodGet, "/", http.StatusSeeOther, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, passed := okHandler()
			req := httptest.NewRequest(tc.method, tc.target, nil)
			rec := httptest.NewRecorder()

			middleware.RequireAuthFlexible(h).ServeHTTP(rec, req)

			if *passed {
				t.Fatal("request reached the next handler without credentials")
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if loc := rec.Header().Get("Location"); tc.wantPage != (loc == "/login") {
				t.Fatalf("Location=%q wantPage=%v", loc, tc.wantPage)
			}
		})
	}
}

func TestRequireAuthFlexible_PublicPathsPassThrough(t *testing.T) {
	cases := []string{
		"/login",
		"/api/v1/login",
		"/setup",
		"/api/v1/setup",
		"/api/v1/get-password-hint",
		"/static/css/app.css",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			h, passed := okHandler()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()

			middleware.RequireAuthFlexible(h).ServeHTTP(rec, req)

			if !*passed {
				t.Fatalf("%s must not require authentication (status %d)", target, rec.Code)
			}
		})
	}
}

// Pins the CURRENT loosness: public-path matching uses strings.HasPrefix,
// so near-miss paths also skip authentication. These endpoints only exist
// at the exact paths today (mux 404s the near-misses at routing level), but
// if public matching is ever tightened to exact paths, these rows must flip.
func TestRequireAuthFlexible_HasPrefixPublicPathLooseness(t *testing.T) {
	cases := []string{
		"/setupX",
		"/api/v1/setup-anything",
		"/loginX",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			h, passed := okHandler()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()

			middleware.RequireAuthFlexible(h).ServeHTTP(rec, req)

			if !*passed {
				t.Fatalf("%s unexpectedly blocked; HasPrefix behavior changed", target)
			}
		})
	}
}

func TestRequireAuthFlexible_ValidBearer(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	plain, _, err := auth.GenerateApiToken(7, "ci-token", 0, "")
	if err != nil {
		t.Fatalf("GenerateApiToken: %v", err)
	}

	var gotAdminID uint
	var gotAuthType string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAdminID, _ = middleware.GetAdminIDFromContext(r)
		gotAuthType, _ = middleware.GetAuthTypeFromContext(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-servers", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec := httptest.NewRecorder()

	middleware.RequireAuthFlexible(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotAdminID != 7 {
		t.Fatalf("context admin id = %d, want 7", gotAdminID)
	}
	if gotAuthType != "bearer" {
		t.Fatalf("context auth type = %q, want bearer", gotAuthType)
	}
}

func TestRequireAuthFlexible_BearerErrors(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	revoked, revokedRec, err := auth.GenerateApiToken(7, "revoked", 0, "")
	if err != nil {
		t.Fatalf("GenerateApiToken: %v", err)
	}
	if err := auth.RevokeApiToken(revokedRec.ID, 7); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	cases := []struct {
		name       string
		bearer     string
		withCookie bool
	}{
		{"garbage token", "Bearer nonsense", false},
		{"empty bearer", "Bearer ", false},
		{"revoked token", "Bearer " + revoked, false},
		{"garbage bearer with valid session (no downgrade)", "Bearer nonsense", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, passed := okHandler()
			var req *http.Request
			if tc.withCookie {
				req = httpsess.AuthedRequest(t, http.MethodGet, "/api/v1/list-servers", 3)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/api/v1/list-servers", nil)
			}
			req.Header.Set("Authorization", tc.bearer)
			rec := httptest.NewRecorder()

			middleware.RequireAuthFlexible(h).ServeHTTP(rec, req)

			if *passed || rec.Code != http.StatusUnauthorized {
				t.Fatalf("passed=%v status=%d, want blocked 401", *passed, rec.Code)
			}
		})
	}
}

func TestRequireAuthFlexible_SessionAuth(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	var gotAdminID uint
	var gotAuthType string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAdminID, _ = middleware.GetAdminIDFromContext(r)
		gotAuthType, _ = middleware.GetAuthTypeFromContext(r)
		w.WriteHeader(http.StatusOK)
	})

	req := httpsess.AuthedRequest(t, http.MethodGet, "/api/v1/list-servers", 42)
	rec := httptest.NewRecorder()

	middleware.RequireAuthFlexible(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotAdminID != 42 {
		t.Fatalf("context admin id = %d, want 42", gotAdminID)
	}
	if gotAuthType != "session" {
		t.Fatalf("context auth type = %q, want session", gotAuthType)
	}
}

// A session marked authenticated but without an admin id must not pass —
// guards the "session valid but no admin id" fallthrough branch.
func TestRequireAuthFlexible_SessionWithoutAdminID(t *testing.T) {
	httpsess.InitTestStore(t)

	emptyReq := httptest.NewRequest(http.MethodGet, "/", nil)
	recSave := httptest.NewRecorder()
	sess, err := middleware.Store.New(emptyReq, "auth")
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	sess.Values["authenticated"] = true // deliberately no admin_id
	if err := sess.Save(emptyReq, recSave); err != nil {
		t.Fatalf("save: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-servers", nil)
	for _, c := range recSave.Result().Cookies() {
		req.AddCookie(c)
	}
	h, passed := okHandler()
	rec := httptest.NewRecorder()

	middleware.RequireAuthFlexible(h).ServeHTTP(rec, req)

	if *passed || rec.Code != http.StatusUnauthorized {
		t.Fatalf("passed=%v status=%d, want blocked 401", *passed, rec.Code)
	}
}

func TestContextHelpersAbsent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if id, ok := middleware.GetAdminIDFromContext(req); ok || id != 0 {
		t.Fatalf("expected absent admin id, got %d/%v", id, ok)
	}
	if at, ok := middleware.GetAuthTypeFromContext(req); ok || at != "" {
		t.Fatalf("expected absent auth type, got %q/%v", at, ok)
	}
}
