// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"strings"
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/testutil"
)

func cptr[T any](v T) *T { return &v }

func seedClientServer(t *testing.T) domain.Server {
	t.Helper()
	srv := domain.Server{
		InterfaceName: "wg0",
		Address:       "10.8.0.1/24",
		Port:          51820,
		PublicKey:     "serverpubKEY",
		PrivateKey:    "serverprivKEY",
		WANAddress:    "203.0.113.7",
	}
	if err := database.DB.Create(&srv).Error; err != nil {
		t.Fatalf("seed server: %v", err)
	}
	return srv
}

func validClient(srv domain.Server) domain.Client {
	return domain.Client{
		Name:        "phone",
		PublicKey:   "clientPUBkey",
		PrivateKey:  "clientPRIVkey",
		AddressCidr: "10.8.0.2/24",
		AllowedIps:  "10.8.0.0/24",
		ServerID:    srv.ID,
	}
}

func TestClientValidate(t *testing.T) {
	testutil.NewTestDB(t)
	srv := seedClientServer(t)

	cases := []struct {
		name   string
		mutate func(*domain.Client)
	}{
		{"empty name", func(c *domain.Client) { c.Name = "  " }},
		{"illegal name chars", func(c *domain.Client) { c.Name = "ph0ne!" }},
		{"empty public key", func(c *domain.Client) { c.PublicKey = "" }},
		{"empty private key", func(c *domain.Client) { c.PrivateKey = "  " }},
		{"space in public key", func(c *domain.Client) { c.PublicKey = "pub key" }},
		{"space in private key", func(c *domain.Client) { c.PrivateKey = "priv key" }},
		{"empty address cidr", func(c *domain.Client) { c.AddressCidr = "" }},
		{"invalid address cidr", func(c *domain.Client) { c.AddressCidr = "10.8.0.2" }},
		{"negative keepalive", func(c *domain.Client) { c.Keepalive = -1 }},
		{"keepalive over 600", func(c *domain.Client) { c.Keepalive = 601 }},
		{"same address as server", func(c *domain.Client) { c.AddressCidr = "10.8.0.1/24" }},
		{"empty allowed ips", func(c *domain.Client) { c.AllowedIps = "  " }},
		{"invalid allowed ips entry", func(c *domain.Client) { c.AllowedIps = "10.8.0.0/24,garbage" }},
		{"too many dns servers", func(c *domain.Client) { c.DnsServers = "8.8.8.8,1.1.1.1,9.9.9.9" }},
		{"invalid dns server", func(c *domain.Client) { c.DnsServers = "not-an-ip" }},
		{"invalid exposed lan entry", func(c *domain.Client) { c.ExposedLans = cptr("192.168.1.0/24,oops/33") }},
		{"unknown server", func(c *domain.Client) { c.ServerID = 999 }},
		// #19: newline injection into the generated [Peer] block
		{"private key newline injection", func(c *domain.Client) { c.PrivateKey = "k3y\nPostUp=evil.sh" }},
		{"public key newline injection", func(c *domain.Client) { c.PublicKey = "k3y\nPostUp=evil.sh" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validClient(srv)
			tc.mutate(&c)
			if err := Validate(&c); err == nil {
				t.Fatalf("expected rejection (%s)", tc.name)
			}
		})
	}

	t.Run("valid client accepted", func(t *testing.T) {
		c := validClient(srv)
		c.Keepalive = 0 // 0 is allowed; only negative and >600 are not
		c.DnsServers = "8.8.8.8, 1.1.1.1"
		c.ExposedLans = cptr("192.168.1.0/24, 10.10.0.0/16")
		if err := Validate(&c); err != nil {
			t.Fatalf("valid client rejected: %v", err)
		}
	})

	t.Run("keepalive boundary 600 accepted", func(t *testing.T) {
		c := validClient(srv)
		c.Keepalive = 600
		if err := Validate(&c); err != nil {
			t.Fatalf("keepalive=600 rejected: %v", err)
		}
	})
}

func TestClientValidate_UniquenessWithinServer(t *testing.T) {
	testutil.NewTestDB(t)
	srv := seedClientServer(t)

	first := validClient(srv)
	if err := database.DB.Create(&first).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}

	t.Run("duplicate name same server", func(t *testing.T) {
		c := validClient(srv)
		c.AddressCidr = "10.8.0.3/24" // name stays "phone"
		err := Validate(&c)
		if err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("duplicate name not rejected: %v", err)
		}
	})

	t.Run("duplicate address same server", func(t *testing.T) {
		c := validClient(srv)
		c.Name = "tablet" // address stays 10.8.0.2/24
		err := Validate(&c)
		if err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("duplicate address not rejected: %v", err)
		}
	})

	t.Run("same name different server allowed", func(t *testing.T) {
		other := domain.Server{
			InterfaceName: "wg1",
			Address:       "10.9.0.1/24",
			Port:          51821,
			PublicKey:     "srv2PUB",
			PrivateKey:    "srv2PRIV",
			WANAddress:    "203.0.113.8",
		}
		if err := database.DB.Create(&other).Error; err != nil {
			t.Fatalf("seed second server: %v", err)
		}
		c := validClient(other) // name "phone" again, but different server
		c.AddressCidr = "10.9.0.2/24"
		if err := Validate(&c); err != nil {
			t.Fatalf("same name on a different server rejected: %v", err)
		}
	})
}

// Client-side mirror of the update contract: ExposedLans (pointer) clears
// when the payload omits/empties it; DnsServers "" clears; keepalive 0
// overwrites (the client UI sends full payloads on every edit).
func TestUpdateClient_OptionalFieldSemantics(t *testing.T) {
	testutil.NewTestDB(t)
	srv := seedClientServer(t)

	old := validClient(srv)
	old.DnsServers = "8.8.8.8"
	old.ExposedLans = cptr("192.168.1.0/24")
	old.Keepalive = 45
	if err := database.DB.Create(&old).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	updated := validClient(srv) // same name/address as old
	updated.ID = old.ID
	updated.DnsServers = ""
	updated.ExposedLans = nil
	updated.Keepalive = 0

	if err := UpdateClient(&old, updated); err != nil {
		t.Fatalf("UpdateClient: %v", err)
	}
	if old.DnsServers != "" {
		t.Fatalf("empty dns must clear, got %q", old.DnsServers)
	}
	if old.ExposedLans != nil {
		t.Fatalf("nil exposedLans must clear, got %q", *old.ExposedLans)
	}
	if old.Keepalive != 0 {
		t.Fatalf("keepalive 0 must overwrite 45, got %d", old.Keepalive)
	}
}
