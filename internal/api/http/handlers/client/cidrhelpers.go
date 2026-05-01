// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"nullguard/internal/pkg/validation"
)

// CidrCheckbox represents one entry in a CIDR checkbox group rendered
// server-side, e.g. for bridge networks or reachable peer LANs.
type CidrCheckbox struct {
	CIDR    string
	Checked bool
}

// buildCidrCheckboxes returns the checkbox list for sourceCsv, marking each
// entry checked if it currently appears in allowedIps.
func buildCidrCheckboxes(sourceCsv, allowedIps string) []CidrCheckbox {
	source := validation.SplitCidrCsv(sourceCsv)
	if len(source) == 0 {
		return nil
	}

	active := map[string]bool{}
	for _, c := range validation.SplitCidrCsv(allowedIps) {
		active[c] = true
	}

	out := make([]CidrCheckbox, 0, len(source))
	for _, c := range source {
		out = append(out, CidrCheckbox{CIDR: c, Checked: active[c]})
	}
	return out
}
