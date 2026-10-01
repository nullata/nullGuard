// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import "sync"

// serverLocks holds one mutex per WireGuard interface name.
//
// All operations that mutate a server's interface — start, stop, restart
// (manual or auto), config regeneration — must run under that interface's
// lock. Without it, concurrent paths (e.g. AutoRestartIfEnabled firing from
// three client handlers while an admin clicks Restart) interleave wg-quick
// down/up calls: "already exists" failures on up, PostUp iptables rules
// applied twice, and concurrent writes to <iface>.conf.
//
// Note this is per-process state: nullGuard is a single-instance app.
var serverLocks sync.Map // interface name -> *sync.Mutex

func serverLock(interfaceName string) *sync.Mutex {
	lock, _ := serverLocks.LoadOrStore(interfaceName, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// WithServerLock runs fn while holding the mutex for the given interface
// name, serializing all interface lifecycle operations. fn's error is
// returned unchanged.
//
// fn must not call WithServerLock for any interface itself (the locks are
// not reentrant, and nesting risks deadlocks).
func WithServerLock(interfaceName string, fn func() error) error {
	lock := serverLock(interfaceName)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}
