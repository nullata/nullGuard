// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

// Package testutil provides shared test infrastructure: an isolated
// in-memory SQLite database wired to the global database.DB.
//
// Tests using these helpers must not run in parallel with each other
// (t.Parallel()), because they temporarily swap process-global state —
// this mirrors how the app itself works.
//
// HTTP/session helpers live in testutil/httpsess to avoid an import cycle
// (middleware depends on repository, so repository tests must not pull in
// middleware).
package testutil

import (
	"fmt"
	"strings"
	"testing"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// NewTestDB opens an isolated in-memory SQLite database, migrates the full
// production model set, and points the global database.DB at it for the
// duration of the test (restored on cleanup).
//
// It mirrors the production SQLite setup — a single connection — so tests
// exercise statement interleaving the same way a deployed sqlite instance
// does. Each test gets its own named in-memory database, so tests never see
// each other's rows.
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	name := strings.NewReplacer("/", "-", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("testutil: open in-memory sqlite: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("testutil: raw db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1) // same as the production sqlite configuration

	if err := db.AutoMigrate(
		&domain.Server{},
		&domain.Client{},
		&domain.Admin{},
		&domain.ApiToken{},
	); err != nil {
		t.Fatalf("testutil: automigrate: %v", err)
	}

	prev := database.DB
	database.DB = db

	t.Cleanup(func() {
		database.DB = prev
		if raw, err := db.DB(); err == nil {
			_ = raw.Close()
		}
	})

	return db
}
