// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"net/http"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/config"
	"nullguard/internal/infrastructure/template"
	serverservice "nullguard/internal/service/server"
)

func CreateServerPageHandler(w http.ResponseWriter, r *http.Request) {
	// fetch data using the service layer
	serverList, err := serverservice.GetFullServerList()
	if err != nil {
		template.ErrorPageHandler(w, "Error fetching server list", http.StatusInternalServerError)
		return
	}

	var defaultServer domain.Server
	var isFirstServer bool = false
	if len(serverList) == 0 {
		isFirstServer = true
		// set default values for the form when no servers exist
		defaultServer, err = serverservice.BuildDefaultServerConf()
		if err != nil {
			template.ErrorPageHandler(w, "Error building default server configuration", http.StatusInternalServerError)
			return
		}
	} else {
		// attempt to auto build defaultServer base on what key values are not present in serverList
		newSuggestedPort, newSuggestedAddress, err := serverservice.FindAvailableServerAttributes(serverList)
		if err != nil {
			newSuggestedAddress = config.GetEnv("WG_SERVER_DEFAULT_ADDR", "")
			newSuggestedPort = config.GetEnvInt("WG_SERVER_DEFAULT_PORT", "")
		}
		defaultServer, err = serverservice.BuildNewServerConf(newSuggestedPort, newSuggestedAddress)
		if err != nil {
			template.ErrorPageHandler(w, "Error building default server configuration", http.StatusInternalServerError)
			return
		}
	}

	data := map[string]interface{}{
		"Title":         "nullguard - Create Server",
		"DefaultServer": defaultServer,
		"IsFirstServer": isFirstServer,
	}

	template.TemplateHandler(w, "templates/create-server.html", data)
}
