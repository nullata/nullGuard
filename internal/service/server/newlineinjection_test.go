// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/testutil"
)

// #19: values written verbatim into <iface>.conf must not contain newlines;
// wg-quick executes the file as root, so an injected "PostUp = ..." line is
// arbitrary root command execution.
func TestServerValidate_NewlineInjectionRejected(t *testing.T) {
	testutil.NewTestDB(t)

	injected := "nice\r\nPostUp = iptables -P FORWARD ACCEPT"
	cases := []struct {
		name   string
		mutate func(*domain.Server)
	}{
		{"comment with LF", func(s *domain.Server) { s.Comment = "line1\nPostUp = evil" }},
		{"comment with CRLF", func(s *domain.Server) { s.Comment = injected }},
		{"comment with CR only", func(s *domain.Server) { s.Comment = "a\rb" }},
		{"private key with embedded newline", func(s *domain.Server) { s.PrivateKey = "k3y\nAddress = 9.9.9.9/32" }},
		{"public key with embedded newline", func(s *domain.Server) { s.PublicKey = "k3y\nPostDown = rm -rf /" }},
		// space-free payload: only the new single-line guard stops this one
		{"private key newline injection, no spaces", func(s *domain.Server) { s.PrivateKey = "k3y\nPostUp=evil.sh" }},
		{"public key newline injection, no spaces", func(s *domain.Server) { s.PublicKey = "k3y\nPostUp=evil.sh" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validServer()
			tc.mutate(&s)
			err := Validate(&s)
			if err == nil {
				t.Fatalf("newline payload accepted (%s) - conf injection possible", tc.name)
			}
			// keys payloads also contain spaces, so the space guard may
			// fire first; what matters is rejection before conf write
			if !strings.Contains(err.Error(), "newlines") && !strings.Contains(err.Error(), "spaces") {
				t.Fatalf("wrong rejection reason: %v", err)
			}
		})
	}

	t.Run("PostUp/PostDown keep multi-line support", func(t *testing.T) {
		// the built-in default template spans three lines; commands are
		// the intended content of these fields and must still validate
		s := validServer()
		s.PostUp = ptr("iptables -A FORWARD -i wg0 -j ACCEPT\niptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE")
		s.PostDown = ptr("iptables -D FORWARD -i wg0 -j ACCEPT")
		if err := Validate(&s); err != nil {
			t.Fatalf("legitimate multi-line PostUp/PostDown rejected: %v", err)
		}
	})
}

// Defense-in-depth check on the writer itself: with validation bypassed
// (DB drift), a stored multi-line comment must still not be able to alter
// the structure of the generated conf beyond the comment itself. Pinned
// here as KNOWN behavior: GenerateServerConfig writes Comment verbatim,
// so the gate is validation, not the writer.
func TestGenerateServerConfig_CommentStaysSingleLineWhenValid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WG_SERVER_CONF_PATH", dir)
	testutil.NewTestDB(t)

	s := validServer()
	s.Comment = "edge router"
	if err := GenerateServerConfig(s); err != nil {
		t.Fatalf("GenerateServerConfig: %v", err)
	}
	out, err := os.ReadFile(filepath.Join(dir, "wg0.conf"))
	if err != nil {
		t.Fatalf("read conf: %v", err)
	}
	conf := string(out)
	if !strings.Contains(conf, "# edge router\nAddress = 10.8.0.1/24") {
		t.Fatalf("comment line malformed in output:\n%s", conf)
	}
	// exactly one comment line, no injected directive lines
	for _, line := range strings.Split(conf, "\n") {
		if strings.HasPrefix(line, "#") && line != "# edge router" {
			t.Fatalf("unexpected extra comment/directive line %q in:\n%s", line, conf)
		}
	}
}
