// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/testutil"
)

// Tests for #18: delete/update of a server must hold the per-interface
// lock across the inactive check AND the row/conf mutation, so a
// concurrent deploy cannot start the interface in between and end up
// running with its .conf deleted underneath it.

// execGate stubs wg execution for the race test: wg-quick `up` signals
// on started, blocks until release is closed, then marks the interface
// live (mirroring real wg-quick bringing the interface up on success).
type execGate struct {
	started chan string
	release chan struct{}
	active  atomic.Value // string: `wg show interfaces` output
}

func (e *execGate) wgQuick(args ...string) (string, error) {
	if args[0] == "up" {
		e.started <- args[1]
		<-e.release
		e.active.Store(ifaceOf(args[1]))
	}
	return "", nil
}

func (e *execGate) wgShowInterfaces() (string, error) { return e.active.Load().(string), nil }

func (e *execGate) install(t *testing.T) {
	t.Helper()
	realQuick, realShow := execWgQuick, execWgShowInterfaces
	execWgQuick, execWgShowInterfaces = e.wgQuick, e.wgShowInterfaces
	t.Cleanup(func() { execWgQuick, execWgShowInterfaces = realQuick, realShow })
}

// The #18 race: delete starts while a deploy's wg-quick up is mid-flight
// (interface still down at the pre-lock check). The old handler shape —
// IsServerActive outside any lock, then cascade-delete row + conf —
// passed that check and deleted the conf of an interface seconds from
// being live. With the check inside the lock, delete must block until
// the deploy completes, observe the now-active interface, refuse with
// ErrServerActive, and leave row + clients + conf untouched.
func TestDeleteServerIfInactiveLocked_BlocksDuringDeployThenRefuses(t *testing.T) {
	testutil.NewTestDB(t)
	dir := t.TempDir()
	t.Setenv("WG_SERVER_CONF_PATH", dir)

	g := &execGate{started: make(chan string, 4), release: make(chan struct{})}
	g.active.Store("")
	g.install(t)

	srv := testServer("wg8")
	createServerRow(t, &srv)
	if err := database.DB.Create(&domain.Client{
		Name: "keepme", PublicKey: "cpub", PrivateKey: "cpriv",
		AddressCidr: "10.240.0.9/32", AllowedIps: "10.240.0.0/24", ServerID: srv.ID,
	}).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}
	confPath := filepath.Join(dir, "wg8.conf")
	if err := os.WriteFile(confPath, []byte("[Interface]\n"), 0600); err != nil {
		t.Fatal(err)
	}

	deployErr := make(chan error, 1)
	go func() { deployErr <- DeployServerLocked(srv) }()
	<-g.started // wg-quick up is now running (and blocked)

	deleteErr := make(chan error, 1)
	go func() { deleteErr <- DeleteServerIfInactiveLocked(srv) }()

	// delete must still be waiting on the lock; the pre-fix code was
	// already past its (unlocked) check and would have deleted by now
	select {
	case err := <-deleteErr:
		t.Fatalf("delete completed (%v) while wg-quick up was in flight — TOCTOU window still open", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(g.release)
	if err := <-deployErr; err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if err := <-deleteErr; !errors.Is(err, ErrServerActive) {
		t.Fatalf("after racing a successful deploy, want ErrServerActive, got %v", err)
	}

	var rows int64
	database.DB.Model(&domain.Server{}).Where("id = ?", srv.ID).Count(&rows)
	if rows != 1 {
		t.Fatal("server row deleted while interface came up")
	}
	var clients int64
	database.DB.Model(&domain.Client{}).Where("server_id = ?", srv.ID).Count(&clients)
	if clients != 1 {
		t.Fatal("clients cascade-deleted while interface came up")
	}
	if _, err := os.Stat(confPath); err != nil {
		t.Fatalf("conf missing under a running interface: %v", err)
	}
}

// Stop ordering also matters after the lock: an inactive interface is
// cascade-deleted cleanly (row, clients, conf all gone).
func TestDeleteServerIfInactiveLocked_InactiveCascades(t *testing.T) {
	testutil.NewTestDB(t)
	dir := t.TempDir()
	t.Setenv("WG_SERVER_CONF_PATH", dir)

	g := &execGate{started: make(chan string, 4), release: make(chan struct{})}
	g.active.Store("") // nothing running
	g.install(t)

	srv := testServer("wg8")
	createServerRow(t, &srv)
	if err := database.DB.Create(&domain.Client{
		Name: "gone", PublicKey: "cpub", PrivateKey: "cpriv",
		AddressCidr: "10.240.0.9/32", AllowedIps: "10.240.0.0/24", ServerID: srv.ID,
	}).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}
	confPath := filepath.Join(dir, "wg8.conf")
	if err := os.WriteFile(confPath, []byte("[Interface]\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := DeleteServerIfInactiveLocked(srv); err != nil {
		t.Fatalf("inactive delete: %v", err)
	}
	var rows int64
	database.DB.Model(&domain.Server{}).Where("id = ?", srv.ID).Count(&rows)
	if rows != 0 {
		t.Fatal("server row survived delete")
	}
	var clients int64
	database.DB.Model(&domain.Client{}).Where("server_id = ?", srv.ID).Count(&clients)
	if clients != 0 {
		t.Fatal("clients survived cascade delete")
	}
	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Fatalf("conf should be deleted, stat: %v", err)
	}
}

// Active interface: update refuses before touching the row or the conf.
func TestUpdateServerIfInactiveLocked_ActiveRefuses(t *testing.T) {
	testutil.NewTestDB(t)
	dir := t.TempDir()
	t.Setenv("WG_SERVER_CONF_PATH", dir)

	g := &execGate{started: make(chan string, 4), release: make(chan struct{})}
	g.active.Store("wg9") // running
	g.install(t)

	old := testServer("wg9")
	old.Comment = "original"
	createServerRow(t, &old)
	confPath := filepath.Join(dir, "wg9.conf")
	if err := os.WriteFile(confPath, []byte("[Interface]\n"), 0600); err != nil {
		t.Fatal(err)
	}

	changed := old
	changed.Port = 51999
	changed.Comment = "rewritten"
	if err := UpdateServerIfInactiveLocked(&old, changed); !errors.Is(err, ErrServerActive) {
		t.Fatalf("want ErrServerActive, got %v", err)
	}

	fresh, err := GetServerByID(int(old.ID))
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if fresh.Port != 51820 || fresh.Comment != "original" {
		t.Fatalf("row mutated despite active refusal: port=%d comment=%q", fresh.Port, fresh.Comment)
	}
	if _, err := os.Stat(confPath); err != nil {
		t.Fatalf("conf deleted despite active refusal: %v", err)
	}
}

// Inactive interface: update applies and the stale conf is removed, all
// under the lock.
func TestUpdateServerIfInactiveLocked_InactiveApplies(t *testing.T) {
	testutil.NewTestDB(t)
	dir := t.TempDir()
	t.Setenv("WG_SERVER_CONF_PATH", dir)

	g := &execGate{started: make(chan string, 4), release: make(chan struct{})}
	g.active.Store("")
	g.install(t)

	old := testServer("wg9")
	old.Comment = "original"
	createServerRow(t, &old)
	confPath := filepath.Join(dir, "wg9.conf")
	if err := os.WriteFile(confPath, []byte("[Interface]\n"), 0600); err != nil {
		t.Fatal(err)
	}

	changed := old
	changed.Comment = "rewritten"
	if err := UpdateServerIfInactiveLocked(&old, changed); err != nil {
		t.Fatalf("inactive update: %v", err)
	}
	fresh, err := GetServerByID(int(old.ID))
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if fresh.Comment != "rewritten" {
		t.Fatalf("comment not updated: %q", fresh.Comment)
	}
	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Fatalf("stale conf should be removed, stat: %v", err)
	}
}

// The delete wrapper and a concurrent restart must never overlap either:
// high-water mark of in-flight wg-quick ops stays 1.
func TestDeleteWrapperSerializesWithRestart(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("wg6") // restart eligible; delete must refuse while active

	srv := testServer("wg6")
	createServerRow(t, &srv)

	var wg sync.WaitGroup
	var deletes, restarts atomic.Int64
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				if err := DeleteServerIfInactiveLocked(srv); errors.Is(err, ErrServerActive) {
					deletes.Add(1)
				}
			} else if err := RestartServerLocked(srv); err == nil {
				restarts.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if got := f.maxConcurrent("wg6"); got > 1 {
		t.Fatalf("wg-quick ops for wg6 overlapped %d-deep (seq: %v)", got, f.sequence())
	}
	if deletes.Load() != 3 {
		t.Fatalf("expected all 3 deletes to hit ErrServerActive, got %d", deletes.Load())
	}
	if restarts.Load() != 3 {
		t.Fatalf("expected 3 successful restarts, got %d (seq: %v)", restarts.Load(), f.sequence())
	}
}
