// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package help

import (
	"log"
	"net/http"
	"regexp"

	"nullguard/internal/infrastructure/template"

	"github.com/gorilla/mux"
)

// keyPattern matches valid cheatsheet keys. Restricting to lowercase
// alphanumerics + dashes prevents lookups against arbitrary template
// names like "base.html" or "header.html".
var keyPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// CheatsheetHandler serves the rendered HTML body of a single cheatsheet
// partial. Returned content is dropped directly into the existing #modal
// by the frontend, so no surrounding chrome is rendered here.
func CheatsheetHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if !keyPattern.MatchString(key) {
		http.NotFound(w, r)
		return
	}

	name := "cheatsheet-" + key
	if !template.LookupTemplate(name) {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := template.RenderTemplate(w, name, nil); err != nil {
		log.Printf("Error rendering cheatsheet %q: %v", key, err)
	}
}
