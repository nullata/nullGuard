// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"strings"
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/testutil"
)

func ptr[T any](v T) *T { return &v }

func validServer() domain.Server {
	return domain.Server{
		InterfaceName: "wg0",
		Address:       "10.8.0.1/24",
		Port:          51820,
		PublicKey:     "pubkeyABC",
		PrivateKey:    "privkeyXYZ",
		WANAddress:    "203.0.113.7",
	}
}

// Server.Validate gates every create/update; these are the user-facing
// contract rows (Linux interface limits, key sanity, port, CSV fields,
// uniqueness).
func TestServerValidate(t *testing.T) {
	testutil.NewTestDB(t)

	// each mutator breaks exactly one rule from the valid baseline
	cases := []struct {
		name  string
		mutate func(*domain.Server)
	}{
		{"empty interface name", func(s *domain.Server) { s.InterfaceName = "  " }},
		{"illegal chars in name", func(s *domain.Server) { s.InterfaceName = "wg/0" }},
		{"16-char name rejected", func(s *domain.Server) { s.InterfaceName = strings.Repeat("a", 16) }},
		{"empty public key", func(s *domain.Server) { s.PublicKey = "" }},
		{"empty private key", func(s *domain.Server) { s.PrivateKey = "  " }},
		{"space in public key", func(s *domain.Server) { s.PublicKey = "pub key" }},
		{"space in private key", func(s *domain.Server) { s.PrivateKey = "priv key" }},
		{"empty address", func(s *domain.Server) { s.Address = "" }},
		{"empty wan address", func(s *domain.Server) { s.WANAddress = "" }},
		{"invalid wan address", func(s *domain.Server) { s.WANAddress = "example.com" }},
		{"invalid address cidr", func(s *domain.Server) { s.Address = "10.8.0.1" }},
		{"port zero", func(s *domain.Server) { s.Port = 0 }},
		{"bad supernet entry", func(s *domain.Server) { s.SupernetCidr = ptr("10.0.0.0/24,garbage") }},
		{"bad bridge network", func(s *domain.Server) { s.BridgeNetworks = ptr("999.999.0.0/16") }},
		{"negative default keepalive", func(s *domain.Server) { s.DefaultKeepalive = ptr(-1) }},
		{"default keepalive over 600", func(s *domain.Server) { s.DefaultKeepalive = ptr(601) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validServer()
			tc.mutate(&s)
			if err := Validate(&s); err == nil {
				t.Fatalf("expected rejection (%s)", tc.name)
			}
		})
	}

	t.Run("15-char name accepted", func(t *testing.T) {
		s := validServer()
		s.InterfaceName = strings.Repeat("b", 15) // Linux interface limit
		if err := Validate(&s); err != nil {
			t.Fatalf("15-char name rejected: %v", err)
		}
	})

	t.Run("valid csv fields and keepalive bounds accepted", func(t *testing.T) {
		s := validServer()
		s.SupernetCidr = ptr("10.0.0.0/24, 172.16.0.0/16")
		s.BridgeNetworks = ptr("192.168.5.0/24")
		s.DefaultKeepalive = ptr(600)
		if err := Validate(&s); err != nil {
			t.Fatalf("valid server rejected: %v", err)
		}
	})
}

func TestServerValidate_UniquenessConflicts(t *testing.T) {
	testutil.NewTestDB(t)

	existing := validServer()
	// direct insert so the ID is populated on our local copy (the service
	// CreateServer takes the struct by value); self-update detection below
	// depends on matching IDs
	if err := database.DB.Create(&existing).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*domain.Server)
	}{
		// each keeps two fields fresh and collides on exactly one
		{"duplicate interface name", func(s *domain.Server) { s.Address = "10.9.0.1/24"; s.Port = 51821 }}, // name stays wg0
		{"duplicate address", func(s *domain.Server) { s.InterfaceName = "wg1"; s.Port = 51821 }},           // address stays 10.8.0.1/24
		{"duplicate port", func(s *domain.Server) { s.InterfaceName = "wg1"; s.Address = "10.9.0.1/24" }},   // port stays 51820
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validServer() // identical to the seeded row before mutation
			tc.mutate(&s)
			err := Validate(&s)
			if err == nil {
				t.Fatalf("expected duplicate %s to be rejected", tc.name)
			}
			if !strings.Contains(err.Error(), "already exists") {
				t.Fatalf("wrong error for %s: %v", tc.name, err)
			}
		})
	}

	t.Run("same server updating itself is not a conflict", func(t *testing.T) {
		s := existing
		if err := Validate(&s); err != nil {
			t.Fatalf("self-update flagged as conflict: %v", err)
		}
	})
}
