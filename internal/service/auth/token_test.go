// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/repository"
	"nullguard/internal/testutil"
)

func TestApiTokenIsValidTruthTable(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	revoked := time.Now().Add(-time.Minute)

	cases := []struct {
		name  string
		token domain.ApiToken
		want  bool
	}{
		{"fresh never-expiring", domain.ApiToken{}, true},
		{"expires in future", domain.ApiToken{ExpiresAt: &future}, true},
		{"expired", domain.ApiToken{ExpiresAt: &past}, false},
		{"revoked", domain.ApiToken{RevokedAt: &revoked}, false},
		{"revoked and expired", domain.ApiToken{RevokedAt: &revoked, ExpiresAt: &past}, false},
		{"revoked but not expired", domain.ApiToken{RevokedAt: &revoked, ExpiresAt: &future}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.token.IsValid(); got != tc.want {
				t.Fatalf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGenerateApiToken_HashesAtRestAndSetsExpiry(t *testing.T) {
	testutil.NewTestDB(t)

	plain, rec, err := GenerateApiToken(1, "ci", 0, "10.0.0.9")
	if err != nil {
		t.Fatalf("GenerateApiToken: %v", err)
	}
	if plain == "" {
		t.Fatal("empty plaintext token returned")
	}
	if rec.ExpiresAt != nil {
		t.Fatalf("expiresInDays=0 must never expire, got %v", *rec.ExpiresAt)
	}

	// at rest: only the sha256 hash is stored, never the plaintext
	stored, err := repository.GetApiTokenByID(rec.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.TokenHash == plain {
		t.Fatal("plaintext token stored in the database")
	}
	sum := sha256.Sum256([]byte(plain))
	if stored.TokenHash != fmt.Sprintf("%x", sum) {
		t.Fatalf("stored hash is not sha256 of the plaintext")
	}
	if stored.CreatedByIP != "10.0.0.9" {
		t.Fatalf("created_by_ip = %q, want 10.0.0.9", stored.CreatedByIP)
	}

	// expiring variant
	_, rec7, err := GenerateApiToken(1, "week", 7, "")
	if err != nil {
		t.Fatalf("GenerateApiToken(7d): %v", err)
	}
	if rec7.ExpiresAt == nil || !rec7.ExpiresAt.After(time.Now()) {
		t.Fatalf("expiresInDays=7 must store a future expiry, got %v", rec7.ExpiresAt)
	}
}

func TestValidateApiToken_Lifecycle(t *testing.T) {
	testutil.NewTestDB(t)

	plain, rec, err := GenerateApiToken(1, "lifecycle", 0, "")
	if err != nil {
		t.Fatalf("GenerateApiToken: %v", err)
	}

	t.Run("valid token authenticates and stamps last used", func(t *testing.T) {
		got, err := ValidateApiToken(plain, "1.2.3.4")
		if err != nil {
			t.Fatalf("ValidateApiToken: %v", err)
		}
		if got.ID != rec.ID {
			t.Fatalf("returned token id %d, want %d", got.ID, rec.ID)
		}
		stored, err := repository.GetApiTokenByID(rec.ID)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if stored.LastUsedIP != "1.2.3.4" || stored.LastUsedAt == nil {
			t.Fatalf("last-used not stamped: ip=%q at=%v", stored.LastUsedIP, stored.LastUsedAt)
		}
	})

	t.Run("garbage token rejected with no side effects", func(t *testing.T) {
		before, _ := repository.ListApiTokensByAdminID(1)
		if _, err := ValidateApiToken("not-a-real-token", "9.9.9.9"); err == nil {
			t.Fatal("expected rejection of unknown token")
		}
		after, _ := repository.ListApiTokensByAdminID(1)
		if len(before) != len(after) {
			t.Fatal("token list changed on rejected auth")
		}
		for _, tok := range after {
			if tok.LastUsedIP == "9.9.9.9" {
				t.Fatal("rejected garbage token touched a stored token's last-used IP")
			}
		}
	})

	t.Run("expired token rejected", func(t *testing.T) {
		expired := time.Now().Add(-time.Minute)
		if err := database.DB.Model(&domain.ApiToken{}).
			Where("id = ?", rec.ID).
			Update("expires_at", &expired).Error; err != nil {
			t.Fatalf("force-expire: %v", err)
		}
		if _, err := ValidateApiToken(plain, "1.2.3.4"); err == nil {
			t.Fatal("expected rejection of expired token")
		}
		// restore: clear expiry so the revoke subtest exercises revocation alone
		if err := database.DB.Model(&domain.ApiToken{}).
			Where("id = ?", rec.ID).
			Update("expires_at", nil).Error; err != nil {
			t.Fatalf("clear expiry: %v", err)
		}
	})

	t.Run("revoked token rejected", func(t *testing.T) {
		if err := RevokeApiToken(rec.ID, 1); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		if _, err := ValidateApiToken(plain, "1.2.3.4"); err == nil {
			t.Fatal("expected rejection of revoked token")
		}
	})
}
