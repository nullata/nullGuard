// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package database

import (
	"testing"
)

func TestUpdateFieldIfChanged(t *testing.T) {
	t.Run("different value overwrites", func(t *testing.T) {
		cur := "old"
		UpdateFieldIfChanged(&cur, "new")
		if cur != "new" {
			t.Fatalf("got %q", cur)
		}
	})

	// The zero value overwrites too: there is no "empty means keep".
	// This is the documented PUT contract (AGENTS.md): send the fields
	// you want to keep.
	t.Run("empty string overwrites", func(t *testing.T) {
		cur := "value"
		UpdateFieldIfChanged(&cur, "")
		if cur != "" {
			t.Fatalf("empty must clear, got %q", cur)
		}
	})

	t.Run("zero int overwrites", func(t *testing.T) {
		cur := 30
		UpdateFieldIfChanged(&cur, 0)
		if cur != 0 {
			t.Fatalf("0 must overwrite 30, got %d", cur)
		}
	})

	t.Run("false overwrites true", func(t *testing.T) {
		cur := true
		UpdateFieldIfChanged(&cur, false)
		if cur {
			t.Fatal("false must overwrite true")
		}
	})
}

func TestUpdatePointerFieldIfChanged(t *testing.T) {
	s := func(v string) *string { return &v }
	i := func(v int) *int { return &v }

	t.Run("nil into nil stays nil", func(t *testing.T) {
		var cur *string
		UpdatePointerFieldIfChanged(&cur, nil)
		if cur != nil {
			t.Fatalf("got %v", *cur)
		}
	})

	// OBSERVED contract, pinned: for pointer fields, nil ASSIGNS nil
	// (clears) because of the `newValue == nil ||` disjunct; only equal
	// non-nil values are skipped. Combined with buildServerObj (JSON ""
	// -> nil pointer), the effective API behavior is: omitting or
	// empty-stringing an optional server field (postUp, supernetCidr,
	// bridgeNetworks, exposedLans) CLEARS it on update. That matches the
	// AGENTS.md documented trap ("empty clears"), though via a different
	// mechanism than the pointer helper's name suggests.
	t.Run("nil clears existing value (observed)", func(t *testing.T) {
		cur := s("stored")
		UpdatePointerFieldIfChanged(&cur, nil)
		if cur != nil {
			t.Fatalf("expected nil to overwrite, got %q", *cur)
		}
	})

	t.Run("non-nil empty string overwrites", func(t *testing.T) {
		cur := s("stored")
		UpdatePointerFieldIfChanged(&cur, s(""))
		if cur == nil || *cur != "" {
			t.Fatalf("ptr(\"\") must overwrite, got %v", cur)
		}
	})

	t.Run("non-nil into nil assigns", func(t *testing.T) {
		var cur *string
		UpdatePointerFieldIfChanged(&cur, s("fresh"))
		if cur == nil || *cur != "fresh" {
			t.Fatalf("got %v", cur)
		}
	})

	t.Run("equal value is a no-op", func(t *testing.T) {
		cur := i(7)
		UpdatePointerFieldIfChanged(&cur, i(7))
		if cur == nil || *cur != 7 {
			t.Fatalf("got %v", cur)
		}
	})
}
