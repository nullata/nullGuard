// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func reqWith(remoteAddr string, headers map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

// Regression test for #17: with no TRUSTED_PROXIES configured (the default),
// X-Forwarded-For / X-Real-IP must be ignored entirely — otherwise any
// client that can reach the port spoofs the recorded audit IPs.
func TestGetClientIP_UntrustedPeerIgnoresForwardedHeaders(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	r := reqWith("203.0.113.7:54321", map[string]string{
		"X-Forwarded-For": "1.2.3.4, 10.0.0.1",
		"X-Real-IP":       "5.6.7.8",
	})
	if got := GetClientIP(r); got != "203.0.113.7" {
		t.Fatalf("got %q, want 203.0.113.7 (headers from an untrusted peer must be ignored)", got)
	}
}

func TestGetClientIP_TrustedProxyXFF(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1, 10.0.0.0/8")

	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			"single entry",
			"127.0.0.1:40000", "203.0.113.9", "203.0.113.9",
		},
		{
			// right-to-left walk: the rightmost non-proxy entry is the one a
			// client cannot forge (client-supplied leftmost entries are skipped
			// only if they are proxies; a client-forged leftmost PUBLIC entry
			// still wins positionally — this is the documented XFF tradeoff —
			// so the meaningful assertion is the proxy-skip on the right)
			"trailing proxy chain",
			"127.0.0.1:40000", "203.0.113.9, 10.1.2.3", "203.0.113.9",
		},
		{
			"all entries are proxies -> fall back to peer",
			"127.0.0.1:40000", "10.1.2.3, 127.0.0.1", "127.0.0.1",
		},
		{
			"unparseable entry stops the walk",
			"127.0.0.1:40000", "bogus, 10.1.2.3", "127.0.0.1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := reqWith(tc.remoteAddr, map[string]string{"X-Forwarded-For": tc.xff})
			if got := GetClientIP(r); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetClientIP_TrustedProxyXRealIP(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "127.0.0.0/8")

	r := reqWith("127.0.0.1:40000", map[string]string{"X-Real-IP": "203.0.113.5"})
	if got := GetClientIP(r); got != "203.0.113.5" {
		t.Fatalf("got %q, want 203.0.113.5", got)
	}
}

// IPv6 regression: the old LastIndex(":") split turned "[::1]:1234" into
// "[::1]" (bracket included) and shredded bare IPv6 addresses.
func TestGetClientIP_IPv6RemoteAddr(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	r := reqWith("[::1]:4321", nil)
	if got := GetClientIP(r); got != "::1" {
		t.Fatalf("got %q, want ::1", got)
	}

	r = reqWith("[2001:db8::42]:1234", nil)
	if got := GetClientIP(r); got != "2001:db8::42" {
		t.Fatalf("got %q, want 2001:db8::42", got)
	}
}

func TestGetClientIP_IPv6TrustedProxy(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "::1")

	r := reqWith("[::1]:4321", map[string]string{"X-Forwarded-For": "2001:db8::7"})
	if got := GetClientIP(r); got != "2001:db8::7" {
		t.Fatalf("got %q, want 2001:db8::7", got)
	}
}

func TestGetClientIP_NoHeadersPlainIPv4(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	r := reqWith("192.168.1.50:9999", nil)
	if got := GetClientIP(r); got != "192.168.1.50" {
		t.Fatalf("got %q, want 192.168.1.50", got)
	}
}

// TRUSTED_PROXIES entries that don't parse are skipped with a log, not fatal,
// and don't disable the valid entries.
func TestGetClientIP_MalformedTrustedProxiesEntriesSkipped(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "not-an-ip, 127.0.0.1/32, 999.999.0.0/16")

	r := reqWith("127.0.0.1:40000", map[string]string{"X-Real-IP": "8.8.8.8"})
	if got := GetClientIP(r); got != "8.8.8.8" {
		t.Fatalf("valid entry must still work despite malformed siblings; got %q", got)
	}
}

func TestGetClientIP_UntrustedIPv6PeerIgnoresXFF(t *testing.T) {
	// no trusted proxies at all: spoofed XFF from a direct IPv6 peer is ignored
	t.Setenv("TRUSTED_PROXIES", "")
	r := reqWith("[2001:db8::6667]:443", map[string]string{
		"X-Forwarded-For": "1.2.3.4, 2001:db8::1",
		"X-Real-IP":       "2001:db8::bad",
	})
	if got := GetClientIP(r); got != "2001:db8::6667" {
		t.Fatalf("got %q, want 2001:db8::6667", got)
	}

	// trusted proxies configured, but the peer is not among them: an IPv6
	// peer matching no entry (exact IPv6 IP differs, IPv4 CIDR cannot
	// contain it) must have its XFF ignored
	t.Setenv("TRUSTED_PROXIES", "2001:db8::1, 10.0.0.0/8")
	r2 := reqWith("[2001:db8::dead]:443", map[string]string{
		"X-Forwarded-For": "8.8.8.8",
	})
	if got := GetClientIP(r2); got != "2001:db8::dead" {
		t.Fatalf("got %q, want 2001:db8::dead (peer not in trusted list)", got)
	}
}
