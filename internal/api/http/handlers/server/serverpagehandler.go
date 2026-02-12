// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"net/http"

	"nullguard/internal/infrastructure/template"
	services "nullguard/internal/service/server"
)

// serves the server configuration page
func ServerPageHandler(w http.ResponseWriter, r *http.Request) {
	// fetch data using the service layer
	serverList, err := services.GetServerList()
	if err != nil {
		template.ErrorPageHandler(w, "Error fetching server list", http.StatusInternalServerError)
		return
	}

	if len(serverList) == 0 {
		// redirect to the "create server" page without query params
		http.Redirect(w, r, "/create-server", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title":   "nullguard - Server",
		"Servers": serverList,
	}

	template.TemplateHandler(w, "templates/server.html", data)
}
