// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"testing"

	"nullguard/internal/infrastructure/database"
	"nullguard/internal/testutil"
)

// End-to-end pin of the AGENTS.md PUT contract:
//   - an empty-string optional field in the payload CLEARS the stored value
//   - non-empty replaces; equal values are no-ops
//
// The mechanism: buildServerObj maps JSON "" to a nil pointer, and
// UpdatePointerFieldIfChanged assigns nil (clears) - see
// database/updatefields_test.go for the helper-level truth table.
func TestUpdateServer_EmptyClearsDocumentedContract(t *testing.T) {
	testutil.NewTestDB(t)

	old := validServer()
	old.Comment = "keep me"
	old.PostUp = ptr("iptables -A FORWARD -i wg0 -j ACCEPT")
	old.SupernetCidr = ptr("10.0.0.0/24")
	old.BridgeNetworks = ptr("192.168.5.0/24")
	old.DefaultKeepalive = ptr(25)
	if err := database.DB.Create(&old).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	// simulate what UpdateServer receives after ConvertRawToServer with
	// empty-string postUp/supernetCidr/bridgeNetworks and defaultKeepAlive
	// sent as "", i.e. nil pointers; comment supplied as "" as well.
	newServer := validServer() // same interface/address/port/keys
	newServer.ID = old.ID
	newServer.Comment = "" // plain field: zero value overwrites
	newServer.PostUp = nil
	newServer.SupernetCidr = nil
	newServer.BridgeNetworks = nil
	newServer.DefaultKeepalive = nil

	if err := UpdateServer(&old, newServer); err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}

	if old.Comment != "" {
		t.Fatalf("empty comment must clear, got %q", old.Comment)
	}
	for name, p := range map[string]*string{
		"PostUp":         old.PostUp,
		"SupernetCidr":   old.SupernetCidr,
		"BridgeNetworks": old.BridgeNetworks,
	} {
		if p != nil {
			t.Fatalf("omitted/empty %s must clear to nil, got %q", name, *p)
		}
	}
	if old.DefaultKeepalive != nil {
		t.Fatalf("empty defaultKeepAlive must clear, got %d", *old.DefaultKeepalive)
	}
	// identity fields survive untouched
	if old.InterfaceName != "wg0" || old.Address != "10.8.0.1/24" {
		t.Fatalf("identity fields mutated: %+v", old)
	}
}

func TestUpdateServer_ValuesReplaceAndEqualIsNoop(t *testing.T) {
	testutil.NewTestDB(t)

	old := validServer()
	old.SupernetCidr = ptr("10.0.0.0/24")
	if err := database.DB.Create(&old).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	newServer := validServer()
	newServer.ID = old.ID
	newServer.SupernetCidr = ptr("172.16.0.0/16")
	newServer.Comment = "new comment"
	if err := UpdateServer(&old, newServer); err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}
	if old.SupernetCidr == nil || *old.SupernetCidr != "172.16.0.0/16" {
		t.Fatalf("supernet not replaced: %v", old.SupernetCidr)
	}
	if old.Comment != "new comment" {
		t.Fatalf("comment not replaced: %q", old.Comment)
	}

	// identical second update must not change anything
	before := *old.SupernetCidr
	if err := UpdateServer(&old, newServer); err != nil {
		t.Fatalf("second UpdateServer: %v", err)
	}
	if old.SupernetCidr == nil || *old.SupernetCidr != before {
		t.Fatalf("equal update mutated value")
	}
}
