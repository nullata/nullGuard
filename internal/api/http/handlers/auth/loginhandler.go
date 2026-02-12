// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package auth

import (
	"log"
	"net/http"
	"nullguard/internal/api/http/middleware"
	"nullguard/internal/infrastructure/template"
)

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	if middleware.IsAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl := template.GetTemplate("login.html")
	if tmpl == nil {
		log.Println("Template not found: login.html")
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
