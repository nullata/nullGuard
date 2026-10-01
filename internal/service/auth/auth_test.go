// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"os"
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
	"nullguard/internal/repository"
	"nullguard/internal/testutil"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt cost 14 is ~1s per hash and worse under -race; tests run the
// whole suite at the minimum cost. Hashing semantics (salted bcrypt,
// verify-by-hash, uniform failure) are identical, and production keeps 14.
func TestMain(m *testing.M) {
	bcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}

func mustCreateAdmin(t *testing.T, username, password string) *domain.Admin {
	t.Helper()
	if err := CreateAdminAccount(username, password, password, nil); err != nil {
		t.Fatalf("CreateAdminAccount(%s): %v", username, err)
	}
	admin, err := repository.GetAdminByUsername(username)
	if err != nil {
		t.Fatalf("fetch admin %s: %v", username, err)
	}
	return admin
}

func TestCreateAdminAccount_StoresBcryptHashNotPlaintext(t *testing.T) {
	testutil.NewTestDB(t)
	const pw = "Str0ngPass"

	admin := mustCreateAdmin(t, "alice", pw)

	if admin.PasswordHash == pw {
		t.Fatal("password stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(pw)); err != nil {
		t.Fatalf("stored hash does not verify against the password: %v", err)
	}
}

func TestCreateAdminAccount_Rejections(t *testing.T) {
	testutil.NewTestDB(t)

	cases := []struct {
		name    string
		user    string
		pass    string
		confirm string
	}{
		{"short username", "ab", "Str0ngPass", "Str0ngPass"},
		{"illegal username chars", "al ice", "Str0ngPass", "Str0ngPass"},
		{"short password", "alice", "Ab1", "Ab1"},
		{"password missing digit", "alice", "StrongPass", "StrongPass"},
		{"password missing upper", "alice", "strongpass1", "strongpass1"},
		{"password missing lower", "alice", "STRONGPASS1", "STRONGPASS1"},
		{"confirmation mismatch", "alice", "Str0ngPass", "Str0ngPars"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CreateAdminAccount(tc.user, tc.pass, tc.confirm, nil); err == nil {
				t.Fatalf("expected rejection for %s", tc.name)
			}
		})
	}

	// none of the rejected attempts may have persisted anything
	exists, err := repository.CheckAdminExists()
	if err != nil {
		t.Fatalf("check exists: %v", err)
	}
	if exists {
		t.Fatal("a rejected CreateAdminAccount persisted an admin")
	}
}

func TestCreateAdminAccount_SecondAccountRefused(t *testing.T) {
	testutil.NewTestDB(t)
	mustCreateAdmin(t, "alice", "Str0ngPass")

	if err := CreateAdminAccount("mallory", "Str0ngPass", "Str0ngPass", nil); err == nil {
		t.Fatal("expected second admin account to be refused")
	}
}

func TestAuthenticateUser_UniformErrorNoEnumeration(t *testing.T) {
	testutil.NewTestDB(t)
	mustCreateAdmin(t, "alice", "Str0ngPass")

	unknownErr := ""
	{
		_, err := AuthenticateUser("nosuchuser", "Str0ngPass")
		if err == nil {
			t.Fatal("expected error for unknown user")
		}
		unknownErr = err.Error()
	}
	wrongPwErr := ""
	{
		_, err := AuthenticateUser("alice", "Wr0ngPass")
		if err == nil {
			t.Fatal("expected error for wrong password")
		}
		wrongPwErr = err.Error()
	}

	// same message for both failure modes: no user-enumeration oracle
	if unknownErr != wrongPwErr {
		t.Fatalf("login errors differ, enabling user enumeration: %q vs %q", unknownErr, wrongPwErr)
	}

	admin, err := AuthenticateUser("alice", "Str0ngPass")
	if err != nil {
		t.Fatalf("correct credentials rejected: %v", err)
	}
	if admin.Username != "alice" {
		t.Fatalf("returned admin = %+v, want alice", admin)
	}
}

func TestChangePassword(t *testing.T) {
	testutil.NewTestDB(t)
	admin := mustCreateAdmin(t, "alice", "Str0ngPass")
	oldHash := admin.PasswordHash

	t.Run("wrong old password rejected, hash unchanged", func(t *testing.T) {
		if err := ChangePassword(admin.ID, "Wr0ngOld1", "N3wStr0ng", "N3wStr0ng", nil); err == nil {
			t.Fatal("expected rejection for wrong old password")
		}
		got, err := repository.GetFirstAdmin()
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if got.PasswordHash != oldHash {
			t.Fatal("hash changed despite rejected change")
		}
	})

	t.Run("confirmation mismatch rejected", func(t *testing.T) {
		if err := ChangePassword(admin.ID, "Str0ngPass", "N3wStr0ng", "N3wStr0nx", nil); err == nil {
			t.Fatal("expected rejection for confirmation mismatch")
		}
	})

	t.Run("foreign admin id unauthorized", func(t *testing.T) {
		if err := ChangePassword(admin.ID+100, "Str0ngPass", "N3wStr0ng", "N3wStr0ng", nil); err == nil {
			t.Fatal("expected unauthorized for mismatched admin id")
		}
	})

	t.Run("success rotates hash and hint", func(t *testing.T) {
		hint := "the new one"
		if err := ChangePassword(admin.ID, "Str0ngPass", "N3wStr0ng", "N3wStr0ng", &hint); err != nil {
			t.Fatalf("ChangePassword: %v", err)
		}
		got, err := repository.GetFirstAdmin()
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if got.PasswordHash == oldHash {
			t.Fatal("hash did not change on successful password change")
		}
		if err := VerifyPassword(got.PasswordHash, "Str0ngPass"); err == nil {
			t.Fatal("old password still verifies after rotation")
		}
		if err := VerifyPassword(got.PasswordHash, "N3wStr0ng"); err != nil {
			t.Fatalf("new password does not verify: %v", err)
		}
		if got.PasswordHint == nil || *got.PasswordHint != hint {
			t.Fatalf("hint = %v, want %q", got.PasswordHint, hint)
		}
	})
}

// Sanity guard on the shared test DB helper: the global really is swapped
// and isolated per test (nothing from other tests leaks in).
func TestTestDBIsolation(t *testing.T) {
	testutil.NewTestDB(t)

	var count int64
	if err := database.DB.Model(&domain.Admin{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("fresh test db has %d admins, want 0", count)
	}
	mustCreateAdmin(t, "countme", "Str0ngPass")
	if err := database.DB.Model(&domain.Admin{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d after one create, want 1", count)
	}
}
