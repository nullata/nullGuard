// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"nullguard/internal/api/http/middleware"
	"nullguard/internal/api/http/models"
	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/template"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	clientservice "nullguard/internal/service/client"
	serverservice "nullguard/internal/service/server"
)

func SetClientSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// validate the data from the request and look it up to ensure integrity
	// this allows for the server side rendering to be based directly off the pre-validated client via UpdateClientPageHandler
	// if the update-client page is not pre-populated with data on load, an error will be displayed redirecting the user

	var rawData models.RawClientData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in SetClientSessionHandler: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	client, err := clientservice.ConvertRawToClient(rawData)
	if err != nil {
		log.Printf("Error converting raw data to client: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	validClient, err := clientservice.GetClientByIDAndServerID(client)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Could not find requested client", nil)
		return
	}

	// load up server data too
	server, err := serverservice.GetServerByID(int(validClient.ServerID))
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "There was a problem while loading client data", nil)
		return
	}

	// save the client id to the session
	session, _ := middleware.Store.Get(r, "update-client") // err safe to ignore: gorilla returns a valid empty session on decode failure
	session.Values["clientData"] = &validClient            // store a pointer to the validClient object in the session for later retrieval
	session.Values["serverInterface"] = server.InterfaceName
	session.Values["serverAddress"] = server.Address
	session.Values["serverSupernet"] = server.SupernetCidr

	bridgeNetworksRaw := ""
	if server.BridgeNetworks != nil {
		bridgeNetworksRaw = *server.BridgeNetworks
	}
	session.Values["serverBridgeNetworks"] = bridgeNetworksRaw

	if err := session.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Could not prepare client for editing. Please try again", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "", nil)
}

func ClearClientSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	clientservice.ClearClientSessionData(w, r)

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "", nil)
}

func UpdateClientPageHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.Store.Get(r, "update-client")          // err safe to ignore: gorilla returns a valid empty session on decode failure
	clientData, ok := session.Values["clientData"].(*domain.Client) // retrieve clientData from the session as a pointer to domain.Client
	if !ok {
		log.Printf("Failed to retrieve client data from session or invalid type")
		template.ErrorPageHandler(w, "Invalid or missing client data", http.StatusBadRequest)
		return
	}

	serverInterface, ok1 := session.Values["serverInterface"]
	serverAddress, ok2 := session.Values["serverAddress"]
	serverSupernet, ok3 := session.Values["serverSupernet"]
	if !ok1 || !ok2 || !ok3 {
		missingFields := []string{}
		if !ok1 {
			missingFields = append(missingFields, "serverInterface")
		}
		if !ok2 {
			missingFields = append(missingFields, "serverAddress")
		}
		if !ok3 {
			missingFields = append(missingFields, "serverSupernet")
		}
		log.Printf("Missing session data: %s", strings.Join(missingFields, ", "))
		template.ErrorPageHandler(w, fmt.Sprintf("Invalid or missing server data: %s", strings.Join(missingFields, ", ")), http.StatusBadRequest)
		return
	}

	bridgeRaw, _ := session.Values["serverBridgeNetworks"].(string)
	bridges := buildCidrCheckboxes(bridgeRaw, clientData.AllowedIps)

	// fetch live peer-exposed LANs (excluding self) for the reachable-peer-LANs checkbox group
	var peerLanCheckboxes []CidrCheckbox
	if server, err := serverservice.GetServerByID(int(clientData.ServerID)); err == nil {
		if peerLans, err := clientservice.GetPeerExposedLans(server, clientData.ID); err == nil {
			peerLanCheckboxes = buildCidrCheckboxes(strings.Join(peerLans, ","), clientData.AllowedIps)
		}
	}

	// prepare data for rendering
	data := map[string]interface{}{
		"Title":           "nullguard - Update Client",
		"ClientData":      clientData,
		"ServerInterface": serverInterface,
		"ServerAddress":   serverAddress,
		"ServerSupernet":  serverSupernet,
		"BridgeNetworks":  bridges,
		"PeerLans":        peerLanCheckboxes,
	}

	template.TemplateHandler(w, "templates/update-client.html", data)
}
