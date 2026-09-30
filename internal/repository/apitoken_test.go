// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package repository

import (
	"errors"
	"sync"
	"testing"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/testutil"

	"gorm.io/gorm"
)

func createToken(t *testing.T, adminID uint, hash, name string) *domain.ApiToken {
	t.Helper()
	tok := &domain.ApiToken{
		AdminID:   adminID,
		TokenHash: hash,
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := CreateApiToken(tok); err != nil {
		t.Fatalf("create token: %v", err)
	}
	return tok
}

// Regression test for #16: writing token last-used info must never revive a
// revoked token. The old code did a full-row DB.Save from a struct loaded
// before the revoke, which wrote revoked_at back to NULL.
func TestTouchApiTokenLastUsed_DoesNotReviveRevokedToken(t *testing.T) {
	testutil.NewTestDB(t)
	tok := createToken(t, 1, "hash-1", "my-token")

	if err := RevokeApiToken(tok.ID, 1); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// A stale last-used write for a revoked token must be a no-op,
	// never a resurrection.
	if err := TouchApiTokenLastUsed(tok.ID, "5.6.7.8"); err != nil {
		t.Fatalf("touch on revoked token should not error, got: %v", err)
	}

	got, err := GetApiTokenByID(tok.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.RevokedAt == nil {
		t.Fatal("revoked token was revived by TouchApiTokenLastUsed")
	}
	if got.LastUsedIP != "" {
		t.Fatalf("revoked token should not accept last-used writes, got ip %q", got.LastUsedIP)
	}
}

// The last-used columns must still update while a token is valid.
func TestTouchApiTokenLastUsed_WritesOnlyLastUsedColumns(t *testing.T) {
	testutil.NewTestDB(t)
	tok := createToken(t, 1, "hash-2", "active-token")

	if err := TouchApiTokenLastUsed(tok.ID, "1.2.3.4"); err != nil {
		t.Fatalf("touch: %v", err)
	}

	got, err := GetApiTokenByID(tok.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.LastUsedIP != "1.2.3.4" {
		t.Fatalf("last_used_ip = %q, want 1.2.3.4", got.LastUsedIP)
	}
	if got.LastUsedAt == nil {
		t.Fatal("last_used_at was not written")
	}
	if got.RevokedAt != nil {
		t.Fatal("revoked_at must not be touched by last-used update")
	}
	if got.Name != "active-token" || got.AdminID != 1 || got.TokenHash != "hash-2" {
		t.Fatal("last-used update modified unrelated columns")
	}
}

// Regression test for #13: revoking a nonexistent token (or one owned by
// another admin) used to return the always-nil database.DB.Error, so the
// API reported success when nothing was revoked.
func TestRevokeApiToken_UnknownOrForeignIDReturnsNotFound(t *testing.T) {
	testutil.NewTestDB(t)
	own := createToken(t, 1, "hash-own", "mine")
	createToken(t, 2, "hash-other", "someone-elses")

	cases := []struct {
		name    string
		tokenID uint
		adminID uint
	}{
		{"unknown token id", 99999, 1},
		{"token owned by another admin", own.ID, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := RevokeApiToken(tc.tokenID, tc.adminID)
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("RevokeApiToken(%d, %d) = %v, want ErrRecordNotFound", tc.tokenID, tc.adminID, err)
			}
		})
	}
}

func TestRevokeApiToken_ValidIDSucceeds(t *testing.T) {
	testutil.NewTestDB(t)
	tok := createToken(t, 1, "hash-ok", "good-token")

	if err := RevokeApiToken(tok.ID, 1); err != nil {
		t.Fatalf("revoke valid token: %v", err)
	}
	got, err := GetApiTokenByID(tok.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.RevokedAt == nil {
		t.Fatal("revoked_at not set after successful revoke")
	}
	if got.IsValid() {
		t.Fatal("IsValid() true for revoked token")
	}
}

// Interleaves last-used writes with a revoke, the shape of the #16 race.
// The revoke must win deterministically: every writer either lands before
// the revoke (harmless) or is rejected by the revoked_at IS NULL guard —
// and the final state must be revoked.
func TestTouchLastUsed_ConcurrentWithRevoke_TokenStaysRevoked(t *testing.T) {
	testutil.NewTestDB(t)
	tok := createToken(t, 1, "hash-race", "busy-token")

	const writers = 16
	var wg sync.WaitGroup

	stop := make(chan struct{})
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = TouchApiTokenLastUsed(tok.ID, "10.0.0.1")
			}
		}(i)
	}

	time.Sleep(20 * time.Millisecond) // let writers hammer for a moment

	if err := RevokeApiToken(tok.ID, 1); err != nil {
		close(stop)
		wg.Wait()
		t.Fatalf("revoke during concurrent last-used writes: %v", err)
	}

	// keep hammering after the revoke too — these must never revive it
	time.Sleep(20 * time.Millisecond)
	close(stop)
	wg.Wait()

	got, err := GetApiTokenByID(tok.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.RevokedAt == nil {
		t.Fatal("token was revived by a concurrent last-used write (regression of #16)")
	}
}
