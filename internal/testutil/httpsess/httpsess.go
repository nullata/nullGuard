// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

// Package httpsess provides HTTP/session test helpers: a swap-in cookie
// store and builders for requests carrying valid auth-session cookies.
//
// It lives in its own package because middleware depends on repository;
// pulling middleware into the base testutil package would create import
// cycles for repository-level tests.
package httpsess

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nullguard/internal/api/http/middleware"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

// InitTestStore swaps in a cookie store with a fixed test secret so handler
// tests can mint auth sessions. The previous store is restored on cleanup.
// Tests that use it must not run in parallel (global swap).
func InitTestStore(t *testing.T) {
	t.Helper()

	prev := middleware.Store
	middleware.Store = sessions.NewCookieStore([]byte("test-session-secret"))

	t.Cleanup(func() { middleware.Store = prev })
}

// AuthedRequest builds an HTTP request carrying a valid auth session cookie
// for the given admin id, as if the admin were logged in via the web UI.
// Requires InitTestStore to have been called first. Optional urlVars set
// gorilla/mux path variables (e.g. {"id": "42"}).
func AuthedRequest(t *testing.T, method, target string, adminID uint, urlVars ...map[string]string) *http.Request {
	t.Helper()

	if middleware.Store == nil {
		t.Fatal("httpsess: middleware.Store is nil; call InitTestStore(t) first")
	}

	// gorilla/sessions dereferences the request in New/Save, so mint the
	// cookie against an empty throwaway request rather than nil
	emptyReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	sess, err := middleware.Store.New(emptyReq, "auth")
	if err != nil {
		t.Fatalf("httpsess: new session: %v", err)
	}
	sess.Values["authenticated"] = true
	sess.Values["admin_id"] = adminID
	if err := sess.Save(emptyReq, rec); err != nil {
		t.Fatalf("httpsess: save session: %v", err)
	}

	req := httptest.NewRequest(method, target, nil)
	for _, cookie := range rec.Result().Cookies() {
		req.AddCookie(cookie)
	}

	if len(urlVars) > 0 {
		// SetURLVars returns a new request with the vars context attached
		req = mux.SetURLVars(req, urlVars[0])
	}
	return req
}
