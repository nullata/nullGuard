// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/testutil"
)

// fakeExec replaces the wg/wg-quick/ip exec seams for a test and records
// every invocation. It tracks per-interface concurrency so tests can assert
// that lifecycle operations never overlap on the same interface.
type fakeExec struct {
	mu      sync.Mutex
	seq     []string                 // ordered "<op> <target>" records
	active  map[string]int           // interface -> in-flight wg-quick ops
	maxAct  map[string]int           // interface -> high-water mark of in-flight ops
	blockOn map[string]chan struct{} // if set, wg-quick for that iface waits

	activeIfaces atomic.Value // string: space-separated `wg show interfaces` output
	ipDeleteErr  error
}

func newFakeExec() *fakeExec {
	f := &fakeExec{
		active:  map[string]int{},
		maxAct:  map[string]int{},
		blockOn: map[string]chan struct{}{},
	}
	f.activeIfaces.Store("")
	return f
}

// install swaps the package exec seams for the fake and restores them on
// cleanup.
func (f *fakeExec) install(t *testing.T) {
	t.Helper()

	realWgQuick, realIPDelete, realShow := execWgQuick, execIPLinkDelete, execWgShowInterfaces
	execWgQuick = f.wgQuick
	execIPLinkDelete = f.ipLinkDelete
	execWgShowInterfaces = f.wgShowInterfaces

	t.Cleanup(func() {
		execWgQuick, execIPLinkDelete, execWgShowInterfaces = realWgQuick, realIPDelete, realShow
	})
}

// ifaceOf extracts the interface name a wg-quick target refers to:
// either a bare name or a full config path (/some/dir/wg9.conf).
func ifaceOf(target string) string {
	base := filepath.Base(target)
	return strings.TrimSuffix(base, ".conf")
}

func (f *fakeExec) wgQuick(args ...string) (string, error) {
	op, target := args[0], args[1]
	iface := ifaceOf(target)

	f.mu.Lock()
	f.seq = append(f.seq, op+" "+target)
	f.active[iface]++
	if f.active[iface] > f.maxAct[iface] {
		f.maxAct[iface] = f.active[iface]
	}
	block := f.blockOn[iface]
	f.mu.Unlock()

	if block != nil {
		<-block // hold the operation until the test releases it
	}

	// simulate a little work so concurrent callers can actually overlap
	time.Sleep(2 * time.Millisecond)

	f.mu.Lock()
	f.active[iface]--
	f.mu.Unlock()
	return "", nil
}

func (f *fakeExec) ipLinkDelete(iface string) (string, error) {
	f.mu.Lock()
	f.seq = append(f.seq, "ip-link-delete "+iface)
	f.mu.Unlock()
	if f.ipDeleteErr != nil {
		return "fake ip failure", f.ipDeleteErr
	}
	return "", nil
}

func (f *fakeExec) wgShowInterfaces() (string, error) {
	return f.activeIfaces.Load().(string), nil
}

func (f *fakeExec) setActive(ifaces string) { f.activeIfaces.Store(ifaces) }

func (f *fakeExec) maxConcurrent(iface string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.maxAct[iface]
}

func (f *fakeExec) sequence() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.seq...)
}

func (f *fakeExec) countOf(opPrefix, iface string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.seq {
		i := strings.IndexByte(s, ' ')
		if i < 0 {
			continue
		}
		op, target := s[:i], s[i+1:]
		if op == opPrefix && ifaceOf(target) == iface {
			n++
		}
	}
	return n
}

// setupExecTest wires up the test DB + temp config dir + fake exec.
func setupExecTest(t *testing.T) *fakeExec {
	t.Helper()
	testutil.NewTestDB(t)
	t.Setenv("WG_SERVER_CONF_PATH", t.TempDir())
	f := newFakeExec()
	f.install(t)
	return f
}

// createServerRow inserts the server and back-fills its generated ID.
func createServerRow(t *testing.T, srv *domain.Server) {
	t.Helper()
	if err := database.DB.Create(srv).Error; err != nil {
		t.Fatalf("insert server: %v", err)
	}
}

func testServer(iface string) domain.Server {
	return domain.Server{
		InterfaceName: iface,
		Address:       "10.240.0.1/24",
		Port:          51820,
		PrivateKey:    "aAAACAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		PublicKey:     "pubkey",
		WANAddress:    "1.2.3.4",
	}
}

// Regression test for #14: lifecycle operations on the same interface must
// never run concurrently — previously Restart/Deploy/Stop and the
// fire-and-forget auto-restart could interleave wg-quick down/up calls.
func TestLifecycleOps_NeverOverlapOnSameInterface(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("wg7")
	srv := testServer("wg7")
	srv.AutoRestart = true

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 3 {
			case 0:
				_ = RestartServerLocked(srv)
			case 1:
				_ = StopServerLocked(srv)
			default:
				_ = DeployServerLocked(srv)
			}
		}(i)
	}
	wg.Wait()

	if got := f.maxConcurrent("wg7"); got > 1 {
		t.Fatalf("wg-quick ops for wg7 ran up to %d-deep concurrently; per-interface lock failed (seq: %v)", got, f.sequence())
	}
}

// Different interfaces must not serialize on each other's locks — otherwise
// one slow restart would stall every other server.
func TestLifecycleOps_DifferentInterfacesDoNotBlockEachOther(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("wgA wgB")

	releaseA := make(chan struct{})
	f.mu.Lock()
	f.blockOn["wgA"] = releaseA
	f.mu.Unlock()

	doneA := make(chan error, 1)
	go func() { doneA <- StopServerLocked(testServer("wgA")) }()

	// give the wgA op time to enter its blocked section
	time.Sleep(30 * time.Millisecond)

	doneB := make(chan error, 1)
	go func() { doneB <- StopServerLocked(testServer("wgB")) }()

	select {
	case err := <-doneB:
		if err != nil {
			t.Fatalf("wgB stop returned error while wgA was held: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(releaseA)
		t.Fatal("wgB was blocked behind wgA's lock; locks must be per-interface")
	}

	close(releaseA)
	if err := <-doneA; err != nil {
		t.Fatalf("wgA stop returned error: %v", err)
	}
}

func TestRestartServerLocked_RunsStopThenStart(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("wg5")

	if err := RestartServerLocked(testServer("wg5")); err != nil {
		t.Fatalf("restart: %v", err)
	}

	seq := f.sequence()
	if len(seq) != 2 {
		t.Fatalf("expected exactly down+up, got %v", seq)
	}
	if !strings.HasPrefix(seq[0], "down ") || !strings.HasPrefix(seq[1], "up ") {
		t.Fatalf("expected down before up, got %v", seq)
	}
}

func TestRestartServerLocked_NotActiveReturnsErrServerNotActive(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("") // no live interfaces

	err := RestartServerLocked(testServer("wg5"))
	if !errors.Is(err, ErrServerNotActive) {
		t.Fatalf("err = %v, want ErrServerNotActive", err)
	}
	if len(f.sequence()) != 0 {
		t.Fatalf("no wg-quick calls expected when not active, got %v", f.sequence())
	}
}

func TestStopServerLocked_NotActiveReturnsErrServerNotActive(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("")

	err := StopServerLocked(testServer("wg5"))
	if !errors.Is(err, ErrServerNotActive) {
		t.Fatalf("err = %v, want ErrServerNotActive", err)
	}
	if len(f.sequence()) != 0 {
		t.Fatalf("no wg-quick calls expected when not active, got %v", f.sequence())
	}
}

func TestStopServerLocked_WgQuickFailureFallsBackToIPLinkDelete(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("wg5") // still listed as live after the failed down -> fallback triggers

	origWgQuick := execWgQuick
	execWgQuick = func(args ...string) (string, error) {
		if args[0] == "down" {
			return "fake: something went wrong", errors.New("exit status 1")
		}
		return origWgQuick(args...)
	}

	if err := StopServerLocked(testServer("wg5")); err != nil {
		t.Fatalf("stop with fallback: %v", err)
	}
	if n := f.countOf("ip-link-delete", "wg5"); n != 1 {
		t.Fatalf("expected one ip link delete fallback, got %d (seq: %v)", n, f.sequence())
	}
}

// AutoRestartIfEnabled must fetch from the DB, restart when active, and —
// the #14 regression — serialize concurrent auto-restarts on the same
// interface.
func TestAutoRestartIfEnabled_ConcurrentCallsSerialize(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("wg3")

	srv := testServer("wg3")
	srv.AutoRestart = true
	createServerRow(t, &srv)

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			AutoRestartIfEnabled(int(srv.ID))
		}()
	}
	wg.Wait()

	if got := f.maxConcurrent("wg3"); got > 1 {
		t.Fatalf("auto-restarts overlapped up to depth %d (seq: %v)", got, f.sequence())
	}
	// every one of the 6 calls must have produced a complete down+up pair
	if n := f.countOf("down", "wg3"); n != 6 {
		t.Fatalf("expected 6 down calls, got %d (seq: %v)", n, f.sequence())
	}
	if n := f.countOf("up", "wg3"); n != 6 {
		t.Fatalf("expected 6 up calls, got %d (seq: %v)", n, f.sequence())
	}
}

func TestAutoRestartIfEnabled_FlagOffDoesNothing(t *testing.T) {
	f := setupExecTest(t)
	f.setActive("wg3")

	srv := testServer("wg3")
	srv.AutoRestart = false
	createServerRow(t, &srv)

	AutoRestartIfEnabled(int(srv.ID))

	if len(f.sequence()) != 0 {
		t.Fatalf("no wg-quick calls expected with AutoRestart=false, got %v", f.sequence())
	}
}
