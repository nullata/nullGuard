// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// DecodeJsonObject caps request bodies at 1 MB via MaxBytesReader: without
// it, any client could force unbounded allocation through any POST endpoint.
func TestDecodeJsonObject_BodySizeCap(t *testing.T) {
	t.Run("small body decodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
		rec := httptest.NewRecorder()
		var v struct {
			Name string `json:"name"`
		}
		if err := DecodeJsonObject(&v, rec, req); err != nil {
			t.Fatalf("small body: %v", err)
		}
		if v.Name != "ok" {
			t.Fatalf("decoded %+v", v)
		}
	})

	t.Run("body over 1MB rejected", func(t *testing.T) {
		// valid JSON, just huge: padding string just over the cap
		body := `{"blob":"` + strings.Repeat("x", (1<<20)+16) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		rec := httptest.NewRecorder()
		var v struct {
			Blob string `json:"blob"`
		}
		if err := DecodeJsonObject(&v, rec, req); err == nil {
			t.Fatal("over-cap body accepted; MaxBytesReader not applied")
		}
	})
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct{ in, want string }{
		{"phone.conf", "phone.conf"},
		{`we"ird\name`, `weirdname`},
		{"line1\r\nline2", "line1line2"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := SanitizeFilename(tc.in); got != tc.want {
			t.Fatalf("SanitizeFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
