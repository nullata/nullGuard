// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// define the regex as a package-level variable
var AllowedNameRegex *regexp.Regexp

func init() {
	// init the regex at runtime
	AllowedNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
}

func ValidateCIDR(cidrStr string) (string, error) {
	ip, _, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return "", fmt.Errorf("Invalid CIDR: %s", cidrStr)
	}
	return ip.String(), nil
}

func ValidateIP(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("Invalid IP: %s", ipStr)
	}
	return nil
}

// SplitCidrCsv splits a comma-separated CIDR string into a slice of trimmed,
// non-empty entries. Returns nil for empty input.
func SplitCidrCsv(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, c := range strings.Split(s, ",") {
		c = strings.TrimSpace(c)
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}
