// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"log"
	"net/http"

	"nullguard/internal/api/http/middleware"
	"nullguard/internal/api/http/models"
	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/template"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	clientservice "nullguard/internal/service/client"
	serverservice "nullguard/internal/service/server"
)

func SetCreateClientSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	var rawData models.RawServerData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in SetCreateClientSessionHandler: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	reqServer, err := serverservice.ConvertRawToServer(rawData)
	if err != nil {
		log.Printf("Error converting raw data to server: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	server, err := serverservice.GetServerByID(int(reqServer.ID))
	if err != nil {
		log.Printf("Error fetching server in SetCreateClientSessionHandler: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Could not load requested server", nil)
		return
	}

	session, _ := middleware.Store.Get(r, "create-client") // err safe to ignore: gorilla returns a valid empty session on decode failure
	session.Values["selected-server"] = server.ID

	if err := session.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "There was a problem preparing selected server data", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "", nil)
}

func CreateClientPageHandler(w http.ResponseWriter, r *http.Request) {
	// fetch data using the service layer
	serverList, err := serverservice.GetServerList()
	if err != nil {
		template.ErrorPageHandler(w, "Error fetching server list", http.StatusInternalServerError)
		return
	}

	var defaultClient domain.Client
	defaultClient, err = clientservice.BuildDefaultClient()
	if err != nil {
		template.ErrorPageHandler(w, "Error building default server configuration", http.StatusInternalServerError)
		return
	}

	session, _ := middleware.Store.Get(r, "create-client") // err safe to ignore: gorilla returns a valid empty session on decode failure
	selectedServerID, ok := session.Values["selected-server"].(uint)
	if !ok {
		selectedServerID = 0
	}

	data := map[string]interface{}{
		"Title":            "nullguard - Create Client",
		"Servers":          serverList,
		"SelectedServerID": selectedServerID,
		"DefaultClient":    defaultClient,
	}

	template.TemplateHandler(w, "templates/create-client.html", data)
}
