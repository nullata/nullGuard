// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nullguard/internal/api/http/middleware"
	"nullguard/internal/domain"
	"nullguard/internal/repository"
	"nullguard/internal/testutil"
	"nullguard/internal/testutil/httpsess"
)

// InitSessionStore must apply the hardening flags the app promises:
// HttpOnly, SameSite=Lax, configured MaxAge, Secure only via COOKIE_SECURE.
// (EnforceSecureCookies mutates Store.Options behind a sync.Once and is
// deliberately untested — documented limitation, see unit-test-plan.md.)
func TestInitSessionStore_CookieFlags(t *testing.T) {
	prev := middleware.Store
	t.Cleanup(func() { middleware.Store = prev })

	t.Setenv("SESSION_SECRET_KEY", "unit-test-secret")
	t.Setenv("SESSION_MAX_AGE", "120")
	t.Setenv("COOKIE_SECURE", "false")

	middleware.InitSessionStore()

	if middleware.Store == nil {
		t.Fatal("store not initialized")
	}
	opts := middleware.Store.Options
	if opts == nil {
		t.Fatal("store options nil")
	}
	if !opts.HttpOnly {
		t.Fatal("cookie must be HttpOnly")
	}
	if opts.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want Lax", opts.SameSite)
	}
	if opts.MaxAge != 120 {
		t.Fatalf("MaxAge = %d, want 120 (SESSION_MAX_AGE)", opts.MaxAge)
	}
	if middleware.GetSessionMaxAge() != 120 {
		t.Fatalf("GetSessionMaxAge() = %d, want 120", middleware.GetSessionMaxAge())
	}
	if opts.Secure {
		t.Fatal("Secure must be false when COOKIE_SECURE=false")
	}
	if opts.Path != "/" {
		t.Fatalf("Path = %q, want /", opts.Path)
	}

	// COOKIE_SECURE=true must flip Secure
	t.Setenv("COOKIE_SECURE", "true")
	middleware.InitSessionStore()
	if !middleware.Store.Options.Secure {
		t.Fatal("Secure must be true when COOKIE_SECURE=true")
	}

	// invalid/absent max age falls back to the 3600 default
	t.Setenv("SESSION_MAX_AGE", "junk")
	middleware.InitSessionStore()
	if middleware.GetSessionMaxAge() != 3600 {
		t.Fatalf("GetSessionMaxAge() = %d, want default 3600", middleware.GetSessionMaxAge())
	}
}

func TestIsAuthenticated(t *testing.T) {
	httpsess.InitTestStore(t)

	t.Run("valid session cookie", func(t *testing.T) {
		req := httpsess.AuthedRequest(t, http.MethodGet, "/server", 1)
		if !middleware.IsAuthenticated(req) {
			t.Fatal("valid session not recognized")
		}
	})

	t.Run("no cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/server", nil)
		if middleware.IsAuthenticated(req) {
			t.Fatal("cookieless request reported authenticated")
		}
	})

	t.Run("tampered cookie", func(t *testing.T) {
		req := httpsess.AuthedRequest(t, http.MethodGet, "/server", 1)
		for _, c := range req.Cookies() {
			req.Header.Set("Cookie", c.Name+"=totally-invalid-value")
		}
		if middleware.IsAuthenticated(req) {
			t.Fatal("tampered cookie reported authenticated")
		}
	})

	t.Run("session without authenticated flag", func(t *testing.T) {
		emptyReq := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		sess, err := middleware.Store.New(emptyReq, "auth")
		if err != nil {
			t.Fatalf("new: %v", err)
		}
		sess.Values["admin_id"] = uint(5) // authenticated deliberately missing
		if err := sess.Save(emptyReq, rec); err != nil {
			t.Fatalf("save: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/server", nil)
		for _, c := range rec.Result().Cookies() {
			req.AddCookie(c)
		}
		if middleware.IsAuthenticated(req) {
			t.Fatal("session lacking authenticated=true reported authenticated")
		}
	})
}

func TestRequireSetup(t *testing.T) {
	testutil.NewTestDB(t)
	httpsess.InitTestStore(t)

	t.Run("no admin yet redirects everything but setup/static", func(t *testing.T) {
		for _, target := range []string{"/", "/server", "/api/v1/list-servers"} {
			h, passed := okHandler()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			middleware.RequireSetup(h).ServeHTTP(rec, req)
			if *passed || rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/setup" {
				t.Fatalf("%s: passed=%v status=%d loc=%q, want redirect to /setup",
					target, *passed, rec.Code, rec.Header().Get("Location"))
			}
		}
	})

	t.Run("setup and static paths pass even before an admin exists", func(t *testing.T) {
		for _, target := range []string{"/setup", "/api/v1/setup", "/static/app.css"} {
			h, passed := okHandler()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			middleware.RequireSetup(h).ServeHTTP(rec, req)
			if !*passed {
				t.Fatalf("%s must pass through RequireSetup", target)
			}
		}
	})

	t.Run("once an admin exists everything passes", func(t *testing.T) {
		if err := repository.CreateAdmin(&domain.Admin{
			Username:     "admin",
			PasswordHash: "not-a-real-hash",
		}); err != nil {
			t.Fatalf("seed admin: %v", err)
		}
		for _, target := range []string{"/", "/server", "/api/v1/list-servers"} {
			h, passed := okHandler()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			middleware.RequireSetup(h).ServeHTTP(rec, req)
			if !*passed {
				t.Fatalf("%s must pass once admin exists", target)
			}
		}
	})
}
