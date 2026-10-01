// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"encoding/json"
	"testing"

	"nullguard/internal/api/http/models"
)

// ValidateDeployData gates deploy/restart/stop (via validateAndGetServer)
// and delete. Its callers dereference the returned pointer unconditionally,
// so every rejection must come back as an error, never (nil, nil) (#21).
func TestValidateDeployData(t *testing.T) {
	num := func(s string) json.Number { return json.Number(s) }

	cases := []struct {
		name    string
		data    models.DeployData
		wantID  int
		wantErr bool
	}{
		{"valid pair", models.DeployData{ID: num("7"), InterfaceName: "wg0"}, 7, false},
		{"whitespace name trimmed then accepted", models.DeployData{ID: num("1"), InterfaceName: " wg0 "}, 1, false},
		{"empty interface name", models.DeployData{ID: num("7"), InterfaceName: "  "}, 0, true},
		{"missing id", models.DeployData{InterfaceName: "wg0"}, 0, true},
		{"zero id", models.DeployData{ID: num("0"), InterfaceName: "wg0"}, 0, true},
		{"negative id", models.DeployData{ID: num("-3"), InterfaceName: "wg0"}, 0, true},
		{"non-numeric id", models.DeployData{ID: num("abc"), InterfaceName: "wg0"}, 0, true},
		{"float id", models.DeployData{ID: num("1.5"), InterfaceName: "wg0"}, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateDeployData(tc.data)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got pointer %v (callers would nil-deref)", got)
				}
				if got != nil {
					t.Fatalf("error case must return nil pointer, got %v", *got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil || *got != tc.wantID {
				t.Fatalf("got %v, want %d", got, tc.wantID)
			}
		})
	}
}
