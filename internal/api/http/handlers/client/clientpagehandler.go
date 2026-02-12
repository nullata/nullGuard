// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"nullguard/internal/api/http/models"
	"nullguard/internal/infrastructure/template"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	clientservice "nullguard/internal/service/client"
	serverservice "nullguard/internal/service/server"
)

func ClientPageHandler(w http.ResponseWriter, r *http.Request) {
	// if the simple server list len is == 0 then display the page but on page load use jq to check the number of servers
	// and display a modal to configure a server first with an ok button redirecting to /server
	serverList, err := serverservice.GetServerList()
	if err != nil {
		template.ErrorPageHandler(w, "Error fetching server list", http.StatusInternalServerError)
		return
	}

	if len(serverList) > 0 {
		// fetch data using the service layer
		clientCount, err := clientservice.GetAllClientsCount()
		if err != nil {
			template.ErrorPageHandler(w, "Error loading clients", http.StatusInternalServerError)
			return
		}

		if clientCount == 0 {
			// redirect to the "create server" page without query params
			http.Redirect(w, r, "/create-client", http.StatusSeeOther)
			return
		}
	}

	data := map[string]interface{}{
		"Title":   "nullguard - Client",
		"Mode":    "edit",
		"Servers": serverList,
	}

	template.TemplateHandler(w, "templates/client.html", data)
}

func BuildClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	var rawData models.RawServerData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in BuildClient: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return
	}

	serverID, err := httputil.ConvertJsonNumToIntPtr(rawData.ID)
	if err != nil {
		log.Printf("Error converting raw data to server: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	server, err := serverservice.GetServerByID(*serverID)
	if err != nil {
		log.Printf("Error fetching server: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Error fetching server", nil)
		return
	}

	keepalive := json.Number(strconv.Itoa(*server.DefaultKeepalive))
	subnetCidr, ipNet, err := clientservice.GetSubnetBaseCIDR(server.Address)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "There was a problem determining the server subnet", nil)
		return
	}

	clientCidrAddress, err := clientservice.FindNextAvailableClientCidrAddress(server, ipNet)
	if err != nil {
		log.Printf("Error determining client ip address: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "There was a problem determining the client IP address", nil)
		return
	}

	supernetCidr := ""
	if server.SupernetCidr != nil {
		supernetCidr = *server.SupernetCidr
	}

	serverDns := strings.Split(server.Address, "/")[0]

	clientData := models.RawClientData{
		AddressCidr:        clientCidrAddress,
		AllowedIps:         subnetCidr,
		Keepalive:          keepalive,
		ServerSupernetCidr: supernetCidr,
	}

	responseData := map[string]interface{}{
		"clientData": clientData,
		"serverDns":  serverDns,
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Success", responseData)
}
