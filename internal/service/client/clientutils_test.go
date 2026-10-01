// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/service/server"
	"nullguard/internal/testutil"
)

func TestGetSubnetBaseCIDR(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"10.8.0.1/24", "10.8.0.0/24"},
		{"10.0.0.5/24", "10.0.0.0/24"}, // host bits normalized away
		{"192.168.4.77/16", "192.168.0.0/16"},
		{"10.8.0.0/24", "10.8.0.0/24"}, // already a base
	}
	for _, tc := range cases {
		got, ipNet, err := GetSubnetBaseCIDR(tc.input)
		if err != nil {
			t.Fatalf("GetSubnetBaseCIDR(%q): %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("GetSubnetBaseCIDR(%q) = %q, want %q", tc.input, got, tc.want)
		}
		if ipNet.String() != tc.want {
			t.Fatalf("returned ipNet %v, want %v", ipNet, tc.want)
		}
	}

	if _, _, err := GetSubnetBaseCIDR("garbage"); err == nil {
		t.Fatal("garbage input accepted")
	}
}

func seedUtilServer(t *testing.T, name, address string, port int) domain.Server {
	t.Helper()
	srv := domain.Server{
		InterfaceName: name,
		Address:       address,
		Port:          port,
		PublicKey:     "srvPUB",
		PrivateKey:    "srvPRIV",
		WANAddress:    "203.0.113.7",
	}
	if err := database.DB.Create(&srv).Error; err != nil {
		t.Fatalf("seed server: %v", err)
	}
	return srv
}

func seedClientAddr(t *testing.T, srvID uint, name, cidr string) {
	t.Helper()
	c := domain.Client{
		Name: name, PublicKey: name + "-pub", PrivateKey: name + "-priv",
		AddressCidr: cidr, AllowedIps: "10.8.0.0/24", ServerID: srvID,
	}
	if err := database.DB.Create(&c).Error; err != nil {
		t.Fatalf("seed client %s: %v", name, err)
	}
}

func TestFindNextAvailableClientCidrAddress(t *testing.T) {
	testutil.NewTestDB(t)
	srv := seedUtilServer(t, "wg0", "10.8.0.1/24", 51820)
	_, ipNet, _ := GetSubnetBaseCIDR(srv.Address)

	t.Run("empty subnet skips network and server ip", func(t *testing.T) {
		got, err := FindNextAvailableClientCidrAddress(srv, ipNet)
		if err != nil {
			t.Fatalf("next address: %v", err)
		}
		// 10.8.0.0 is the network address and 10.8.0.1 the server itself
		if got != "10.8.0.2/32" {
			t.Fatalf("got %q, want 10.8.0.2/32", got)
		}
	})

	t.Run("skips addresses already used by clients", func(t *testing.T) {
		seedClientAddr(t, srv.ID, "c1", "10.8.0.2/32")
		seedClientAddr(t, srv.ID, "c2", "10.8.0.3/32")
		got, err := FindNextAvailableClientCidrAddress(srv, ipNet)
		if err != nil {
			t.Fatalf("next address: %v", err)
		}
		if got != "10.8.0.4/32" {
			t.Fatalf("got %q, want 10.8.0.4/32", got)
		}
	})

	t.Run("exhausted subnet errors instead of wrapping", func(t *testing.T) {
		// /30 has exactly two usable hosts: server .1, client .2 -> none left
		tight := seedUtilServer(t, "wg9", "10.9.0.1/30", 51829)
		_, tightNet, _ := GetSubnetBaseCIDR(tight.Address)
		seedClientAddr(t, tight.ID, "t1", "10.9.0.2/32")

		if got, err := FindNextAvailableClientCidrAddress(tight, tightNet); err == nil {
			t.Fatalf("exhausted /30 returned %q instead of erroring", got)
		}
	})
}

func TestMapClientsToRawData_MergesPeerStatus(t *testing.T) {
	clients := []domain.Client{
		{ID: 1, Name: "online", PublicKey: "PK1", AddressCidr: "10.8.0.2/32", Keepalive: 25},
		{ID: 2, Name: "offline", PublicKey: "PK2", AddressCidr: "10.8.0.3/32", Keepalive: 0},
	}
	statuses := map[string]server.PeerStatus{
		"PK1": {IsConnected: true, LastHandshake: 1234, TransferRx: 100, TransferTx: 200},
		// PK2 deliberately absent -> zero status
	}

	raw := MapClientsToRawData(clients, statuses)
	if len(raw) != 2 {
		t.Fatalf("mapped %d clients, want 2", len(raw))
	}

	on, off := raw[0], raw[1]
	if on.Name != "online" || on.ID.String() != "1" || on.Keepalive.String() != "25" {
		t.Fatalf("client fields not mapped: %+v", on)
	}
	if !on.IsConnected || on.LastHandshake != 1234 || on.TransferRx != 100 || on.TransferTx != 200 {
		t.Fatalf("peer status not merged: %+v", on)
	}
	if off.IsConnected || off.LastHandshake != 0 || off.TransferRx != 0 || off.TransferTx != 0 {
		t.Fatalf("absent peer should have zero status: %+v", off)
	}
}
