// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/testutil"
)

// Golden shape of the config a client imports into the WireGuard app.
// Anything that changes this output breaks every existing client profile.
func TestGenerateClientConfig_Golden(t *testing.T) {
	testutil.NewTestDB(t)
	srv := seedUtilServer(t, "wg0", "10.8.0.1/24", 51820)

	c := domain.Client{
		Name:        "phone",
		PrivateKey:  "CLIENTPRIV",
		AddressCidr: "10.8.0.5/32",
		AllowedIps:  "0.0.0.0/0, ::/0",
		DnsServers:  "8.8.8.8, 1.1.1.1",
		Keepalive:   25,
		ServerID:    srv.ID,
	}

	got, err := GenerateClientConfig(c)
	if err != nil {
		t.Fatalf("GenerateClientConfig: %v", err)
	}
	want := `[Interface]
PrivateKey = CLIENTPRIV
Address = 10.8.0.5/32
DNS = 8.8.8.8, 1.1.1.1

[Peer]
PublicKey = srvPUB
Endpoint = 203.0.113.7:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`
	if got != want {
		t.Fatalf("config drift:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenerateClientConfig_OptionalLinesOmitted(t *testing.T) {
	testutil.NewTestDB(t)
	srv := seedUtilServer(t, "wg1", "10.9.0.1/24", 51821)

	c := domain.Client{
		Name: "laptop", PrivateKey: "PRIV", AddressCidr: "10.9.0.2/32",
		AllowedIps: "10.9.0.0/24", Keepalive: 0, ServerID: srv.ID,
	}
	got, err := GenerateClientConfig(c)
	if err != nil {
		t.Fatalf("GenerateClientConfig: %v", err)
	}
	if strings.Contains(got, "DNS") {
		t.Fatalf("empty DNS must produce no DNS line:\n%s", got)
	}
	if strings.Contains(got, "PersistentKeepalive") {
		t.Fatalf("keepalive 0 must produce no PersistentKeepalive line:\n%s", got)
	}
}

func TestGenerateClientConfig_UnknownServerErrors(t *testing.T) {
	testutil.NewTestDB(t)
	if _, err := GenerateClientConfig(domain.Client{ServerID: 4242}); err == nil {
		t.Fatal("unknown server id accepted")
	}
}

func TestCreateConfigZip_Contents(t *testing.T) {
	testutil.NewTestDB(t)
	srv := seedUtilServer(t, "wg2", "10.10.0.1/24", 51822)
	c := domain.Client{
		Name: "tablet", PrivateKey: "ZPRIV", AddressCidr: "10.10.0.2/32",
		AllowedIps: "10.10.0.0/24", ServerID: srv.ID,
	}

	zipBytes, fileName, err := CreateConfigZip(c)
	if err != nil {
		t.Fatalf("CreateConfigZip: %v", err)
	}
	if fileName == "" {
		t.Fatal("empty zip filename returned")
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("zip unreadable: %v", err)
	}
	names := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		rc.Close()
		names[f.Name] = buf.String()
	}
	conf, ok := names["tablet.conf"]
	if !ok {
		t.Fatalf("missing tablet.conf in zip, got %v", names)
	}
	if !strings.Contains(conf, "PrivateKey = ZPRIV") {
		t.Fatalf("zipped conf wrong:\n%s", conf)
	}
	png, ok := names["tablet-qr.png"]
	if !ok || !strings.HasPrefix(png, "\x89PNG") {
		t.Fatalf("missing/invalid tablet-qr.png in zip")
	}
}
