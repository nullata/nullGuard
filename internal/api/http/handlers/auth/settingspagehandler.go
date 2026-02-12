// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"net/http"
	"nullguard/internal/infrastructure/template"
)

// AdminSettingsPageHandler serves the unified admin settings page
func AdminSettingsPageHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "nullguard - Admin Settings",
	}
	template.TemplateHandler(w, "templates/admin-settings.html", data)
}
