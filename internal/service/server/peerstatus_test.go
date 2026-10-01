// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// stubDump replaces the wg-execution seam for the duration of a test.
func stubDump(t *testing.T, stdout string, err error) {
	t.Helper()
	orig := execWgShowDump
	execWgShowDump = func(string) (string, error) { return stdout, err }
	t.Cleanup(func() { execWgShowDump = orig })
}

func wgDumpLine(pub, pkey, ep, allowed string, handshake, rx, tx int64) string {
	return strings.Join([]string{
		pub, pkey, "(none)", ep, fmt.Sprintf("%d", handshake),
		fmt.Sprintf("%d", rx), fmt.Sprintf("%d", tx), "some comment",
	}, "\t")
}

func TestGetPeerStatuses_Parsing(t *testing.T) {
	now := time.Now().Unix()
	recent := now - 30  // inside the 3-minute freshness window
	stale := now - 4*60 // past it
	header := "public key\tpre-shared key\tendpoint\tallowed ips\thandshake\treceived\ttransferred\tpersistent keepalive"

	dump := strings.Join([]string{
		header,
		wgDumpLine("PEERFRESH", "", "1.2.3.4:5678", "10.8.0.2/32", recent, 1000, 2000),
		wgDumpLine("PEERSTALE", "", "(none)", "10.8.0.3/32", stale, 5, 6),
		wgDumpLine("PEERNEVER", "", "(none)", "10.8.0.4/32", 0, 0, 0),
		"short\tline", // fewer than 7 fields -> skipped
		"",            // blank -> skipped
	}, "\n")

	stubDump(t, dump, nil)
	statuses := GetPeerStatuses("wg0")

	if len(statuses) != 3 {
		t.Fatalf("parsed %d peers, want 3: %+v", len(statuses), statuses)
	}
	fresh := statuses["PEERFRESH"]
	if !fresh.IsConnected {
		t.Fatalf("peer with %ds-old handshake should be connected: %+v", now-recent, fresh)
	}
	if fresh.TransferRx != 1000 || fresh.TransferTx != 2000 || fresh.LastHandshake != recent {
		t.Fatalf("counters mis-parsed: %+v", fresh)
	}
	if staleS := statuses["PEERSTALE"]; staleS.IsConnected {
		t.Fatalf("peer with 4-minute-old handshake must not be connected")
	} else if staleS.LastHandshake != stale {
		t.Fatalf("stale handshake recorded as %d, want %d", staleS.LastHandshake, stale)
	}
	if never := statuses["PEERNEVER"]; never.IsConnected || never.LastHandshake != 0 {
		t.Fatalf("never-handshaked peer mis-parsed: %+v", never)
	}
}

func TestGetPeerStatuses_DegenerateInput(t *testing.T) {
	t.Run("wg command failure yields empty map", func(t *testing.T) {
		stubDump(t, "", fmt.Errorf("interface not present"))
		if got := GetPeerStatuses("wg-missing"); len(got) != 0 {
			t.Fatalf("want empty map, got %+v", got)
		}
	})

	t.Run("empty dump yields empty map", func(t *testing.T) {
		stubDump(t, "", nil)
		if got := GetPeerStatuses("wg0"); len(got) != 0 {
			t.Fatalf("want empty map, got %+v", got)
		}
	})

	t.Run("header-only dump yields empty map", func(t *testing.T) {
		stubDump(t, "public key\tpre-shared key\tendpoint\tallowed ips\thandshake\treceived\ttransferred\tpersistent keepalive\n", nil)
		if got := GetPeerStatuses("wg0"); len(got) != 0 {
			t.Fatalf("want empty map, got %+v", got)
		}
	})

	t.Run("non-numeric counters default to zero not error", func(t *testing.T) {
		line := "PUB\t\t(none)\t10.8.0.2/32\tabc\tNaN\txyz\t"
		stubDump(t, "header\n"+line, nil)
		got := GetPeerStatuses("wg0")
		p, ok := got["PUB"]
		if !ok || p.TransferRx != 0 || p.TransferTx != 0 || p.IsConnected {
			t.Fatalf("garbage counters should zero out, got %+v (ok=%v)", p, ok)
		}
	})
}
