// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package version

import (
	"log"
	"os"
	"strings"
)

// appVersion holds the application version read from the VERSION file at
// startup. Defaults to "dev" so a missing file never breaks startup.
var appVersion = "dev"

// ReadFrom sets the application version from the given file (the repository's
// VERSION file), stripping surrounding whitespace. Call once at startup
// before any template rendering.
func ReadFrom(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: could not read version file %q: %v", path, err)
		return
	}
	appVersion = strings.TrimSpace(string(data))
	log.Printf("Application version: %s", appVersion)
}

// Get returns the application version as read from the VERSION file.
func Get() string {
	return appVersion
}
