// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package validation

import (
	"fmt"
	"net"
	"regexp"
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
