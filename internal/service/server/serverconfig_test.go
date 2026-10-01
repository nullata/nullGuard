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

// Regression test for #15: generated server configs must not contain
// SaveConfig = true. With SaveConfig on, `wg-quick down` persists runtime
// interface state (peers added out-of-band via `wg set`, PSKs) back into
// <iface>.conf, letting hand-added peers silently survive on disk.
func TestGenerateServerConfig_NoSaveConfig(t *testing.T) {
	testutil.NewTestDB(t)
	dir := t.TempDir()
	t.Setenv("WG_SERVER_CONF_PATH", dir)

	srv := testServer("wg9")
	srv.Comment = "keep me"
	if err := GenerateServerConfig(srv); err != nil {
		t.Fatalf("GenerateServerConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "wg9.conf"))
	if err != nil {
		t.Fatalf("read generated conf: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "SaveConfig") {
		t.Fatalf("generated conf still contains SaveConfig:\n%s", content)
	}
	// sanity: the rest of the interface section is intact
	for _, want := range []string{"[Interface]", "# keep me", "Address = 10.240.0.1/24", "ListenPort = 51820", "PrivateKey = "} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated conf missing %q:\n%s", want, content)
		}
	}
}

// Peers from the DB must still be rendered after the SaveConfig removal,
// including exposed-LAN expansion and per-client keepalive.
func TestGenerateServerConfig_PeersStillRendered(t *testing.T) {
	db := testutil.NewTestDB(t)
	dir := t.TempDir()
	t.Setenv("WG_SERVER_CONF_PATH", dir)

	srv := testServer("wg8")
	if err := db.Create(&srv).Error; err != nil {
		t.Fatalf("insert server: %v", err)
	}

	exposed := "192.168.5.0/24,10.10.0.0/16"
	ka := 25
	client := domain.Client{
		ServerID:    srv.ID,
		Name:        "laptop",
		PublicKey:   "clientpubkey",
		AddressCidr: "10.240.0.5/32",
		AllowedIps:  "10.240.0.5/32",
		ExposedLans: &exposed,
		Keepalive:   ka,
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatalf("insert client: %v", err)
	}

	if err := GenerateServerConfig(srv); err != nil {
		t.Fatalf("GenerateServerConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "wg8.conf"))
	if err != nil {
		t.Fatalf("read generated conf: %v", err)
	}
	content := string(data)

	for _, want := range []string{"[Peer]", "# laptop", "PublicKey = clientpubkey", "AllowedIPs = 10.240.0.5/32, 192.168.5.0/24, 10.10.0.0/16", "PersistentKeepalive = 25"} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated conf missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "SaveConfig") {
		t.Fatalf("generated conf still contains SaveConfig:\n%s", content)
	}
}
