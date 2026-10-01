// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package validation

import (
	"reflect"
	"testing"
)

func TestValidateCIDR(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantIP  string
		wantErr bool
	}{
		{"ipv4 with mask", "10.8.0.1/24", "10.8.0.1", false},
		{"ipv6 with mask", "fd00::1/64", "fd00::1", false},
		{"host bits allowed", "10.0.0.5/24", "10.0.0.5", false},
		{"bare ip no mask", "10.8.0.1", "", true},
		{"prefix too long", "10.8.0.1/33", "", true},
		{"garbage", "not-a-cidr", "", true},
		{"empty", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateCIDR(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ValidateCIDR(%q) accepted, returned %q", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateCIDR(%q): %v", tc.input, err)
			}
			if got != tc.wantIP {
				t.Fatalf("ValidateCIDR(%q) = %q, want %q", tc.input, got, tc.wantIP)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		{"10.8.0.1", false},
		{"8.8.8.8", false},
		{"::1", false},
		{"fd00::abcd", false},
		{"999.1.1.1", true},
		{"10.8.0", true},
		{"", true},
		{"localhost", true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if err := ValidateIP(tc.input); (err != nil) != tc.wantErr {
				t.Fatalf("ValidateIP(%q) err = %v, wantErr = %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestSplitCidrCsv(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"skips empty entries", "a, ,b", []string{"a", "b"}},
		{"trims spaces", " 10.0.0.0/24 , 10.1.0.0/24 ", []string{"10.0.0.0/24", "10.1.0.0/24"}},
		{"single", "x", []string{"x"}},
		{"trailing comma", "a,", []string{"a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitCidrCsv(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("SplitCidrCsv(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}

func TestAllowedNameRegex(t *testing.T) {
	valid := []string{"wg0", "my.server-1_x", "A9", "a23456789012345"}
	invalid := []string{"", " ", "wg 0", "wg/0", "wg:0", "wg0!", "wg0\n", "wøg"}
	for _, s := range valid {
		if !AllowedNameRegex.MatchString(s) {
			t.Fatalf("%q must match the allowed-name regex", s)
		}
	}
	for _, s := range invalid {
		if AllowedNameRegex.MatchString(s) {
			t.Fatalf("%q must NOT match the allowed-name regex", s)
		}
	}
}
