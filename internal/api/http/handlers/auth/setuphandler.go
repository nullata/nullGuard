// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"log"
	"net/http"
	"nullguard/internal/infrastructure/template"
	authservice "nullguard/internal/service/auth"
)

func SetupPageHandler(w http.ResponseWriter, r *http.Request) {
	adminExists, err := authservice.CheckAdminExists()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if adminExists {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl := template.GetTemplate("setup.html")
	if tmpl == nil {
		log.Println("Template not found: setup.html")
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
