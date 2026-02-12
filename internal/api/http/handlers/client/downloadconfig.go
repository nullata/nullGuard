// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"log"
	"net/http"
	"strconv"

	"nullguard/internal/domain"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	clientservice "nullguard/internal/service/client"

	"github.com/gorilla/mux"
)

// GetClientConfig returns the WireGuard configuration file for a client
func GetClientConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// get client id and server id from url parameters
	vars := mux.Vars(r)
	clientIDStr := vars["clientId"]
	serverIDStr := vars["serverId"]

	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		log.Printf("Invalid client ID: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid client ID", nil)
		return
	}

	serverID, err := strconv.ParseUint(serverIDStr, 10, 32)
	if err != nil {
		log.Printf("Invalid server ID: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid server ID", nil)
		return
	}

	// fetch the client
	client, err := clientservice.GetClientByIDAndServerID(domain.Client{
		ID:       uint(clientID),
		ServerID: uint(serverID),
	})
	if err != nil {
		log.Printf("Error fetching client: %v", err)
		httputil.SendJSONResponse(w, http.StatusNotFound, constants.StatusError, "Client not found", nil)
		return
	}

	// generate the configuration
	config, err := clientservice.GenerateClientConfig(client)
	if err != nil {
		log.Printf("Error generating config: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to generate configuration", nil)
		return
	}

	// set headers for plain text download
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+httputil.SanitizeFilename(client.Name)+".conf\"")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(config)); err != nil {
		log.Printf("Error writing response: %v", err)
		return
	}
}

// GetClientQRCode returns a QR code PNG image for the client configuration
func GetClientQRCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// get client id and server id from url parameters
	vars := mux.Vars(r)
	clientIDStr := vars["clientId"]
	serverIDStr := vars["serverId"]

	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		log.Printf("Invalid client ID: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid client ID", nil)
		return
	}

	serverID, err := strconv.ParseUint(serverIDStr, 10, 32)
	if err != nil {
		log.Printf("Invalid server ID: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid server ID", nil)
		return
	}

	// fetch the client
	client, err := clientservice.GetClientByIDAndServerID(domain.Client{
		ID:       uint(clientID),
		ServerID: uint(serverID),
	})
	if err != nil {
		log.Printf("Error fetching client: %v", err)
		httputil.SendJSONResponse(w, http.StatusNotFound, constants.StatusError, "Client not found", nil)
		return
	}

	// generate the qr code
	qrCode, err := clientservice.GenerateQRCode(client)
	if err != nil {
		log.Printf("Error generating QR code: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to generate QR code", nil)
		return
	}

	// Set headers for PNG image
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "inline; filename=\""+httputil.SanitizeFilename(client.Name)+"-qr.png\"")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(qrCode); err != nil {
		log.Printf("Error writing response: %v", err)
		return
	}
}

// DownloadClientConfig returns a zip file containing the client configuration
func DownloadClientConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return
	}

	// get client id and server id from url parameters
	vars := mux.Vars(r)
	clientIDStr := vars["clientId"]
	serverIDStr := vars["serverId"]

	clientID, err := strconv.ParseUint(clientIDStr, 10, 32)
	if err != nil {
		log.Printf("Invalid client ID: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid client ID", nil)
		return
	}

	serverID, err := strconv.ParseUint(serverIDStr, 10, 32)
	if err != nil {
		log.Printf("Invalid server ID: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid server ID", nil)
		return
	}

	// fetch the client
	client, err := clientservice.GetClientByIDAndServerID(domain.Client{
		ID:       uint(clientID),
		ServerID: uint(serverID),
	})
	if err != nil {
		log.Printf("Error fetching client: %v", err)
		httputil.SendJSONResponse(w, http.StatusNotFound, constants.StatusError, "Client not found", nil)
		return
	}

	// generate the zip file
	zipData, zipFileName, err := clientservice.CreateConfigZip(client)
	if err != nil {
		log.Printf("Error creating zip file: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Failed to create zip file", nil)
		return
	}

	// set headers for zip download
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+httputil.SanitizeFilename(zipFileName)+"\"")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(zipData); err != nil {
		log.Printf("Error writing response: %v", err)
		return
	}
}
