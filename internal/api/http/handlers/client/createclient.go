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
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	clientservice "nullguard/internal/service/client"
	serverservice "nullguard/internal/service/server"
	utils "nullguard/internal/service/wireguard"
)

// CreateClient creates a new WireGuard client for a server
func CreateClient(w http.ResponseWriter, r *http.Request) {
	rawData, ok := httputil.ValidateAndDecodeJSON[models.RawClientData](w, r, http.MethodPost)
	if !ok {
		return
	}

	// server ID is always required
	serverID, err := httputil.ConvertJsonNumToIntPtr(rawData.ServerID)
	if err != nil || serverID == nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "serverId is required", nil)
		return
	}

	// auto-derive optional fields from server when not provided
	needsAddress := strings.TrimSpace(rawData.AddressCidr) == ""
	needsAllowedIps := strings.TrimSpace(rawData.AllowedIps) == ""
	needsKeepalive := rawData.Keepalive == "" || rawData.Keepalive == "0"
	needsKeys := strings.TrimSpace(rawData.PublicKey) == "" || strings.TrimSpace(rawData.PrivateKey) == ""

	if needsAddress || needsAllowedIps || needsKeepalive {
		server, err := serverservice.GetServerByID(*serverID)
		if err != nil {
			log.Printf("Error fetching server: %v", err)
			httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid server ID", nil)
			return
		}

		if needsKeepalive && server.DefaultKeepalive != nil {
			rawData.Keepalive = json.Number(strconv.Itoa(*server.DefaultKeepalive))
		}

		if needsAddress || needsAllowedIps {
			subnetCidr, ipNet, err := clientservice.GetSubnetBaseCIDR(server.Address)
			if err != nil {
				log.Printf("Error determining server subnet: %v", err)
				httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to determine server subnet", nil)
				return
			}

			if needsAllowedIps {
				rawData.AllowedIps = subnetCidr
			}

			if needsAddress {
				clientAddr, err := clientservice.FindNextAvailableClientCidrAddress(server, ipNet)
				if err != nil {
					log.Printf("Error determining client address: %v", err)
					httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to determine client IP address", nil)
					return
				}
				rawData.AddressCidr = clientAddr
			}
		}
	}

	if needsKeys {
		privateKey, publicKey, err := utils.GenerateWgKeys("client")
		if err != nil {
			log.Printf("Error generating client keys: %v", err)
			httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to generate client keys", nil)
			return
		}
		rawData.PrivateKey = strings.TrimSpace(privateKey)
		rawData.PublicKey = strings.TrimSpace(publicKey)
	}

	// convert RawClientData to client using the conversion function
	client, err := clientservice.ConvertRawToClient(*rawData)
	if err != nil {
		log.Printf("Error converting raw data to client: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid data format", nil)
		return
	}

	if err := clientservice.Validate(&client); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return
	}

	if err := clientservice.CreateClient(client); err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "There was a problem creating an entry with the requested server configuration", nil)
		return
	}

	// generate the client config file content
	configContent, err := clientservice.GenerateClientConfig(client)
	if err != nil {
		log.Printf("Client created but failed to generate config: %v", err)
		httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Client created", nil)
		return
	}

	httputil.SendJSONResponse(w, http.StatusOK, constants.StatusSuccess, "Client created", map[string]string{
		"config": configContent,
	})
}
