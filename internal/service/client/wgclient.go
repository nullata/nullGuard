// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nullguard/internal/api/http/middleware"
	"nullguard/internal/api/http/models"
	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/config"
	database "nullguard/internal/infrastructure/database"
	"nullguard/internal/pkg/httputil"
	"nullguard/internal/pkg/validation"
	"nullguard/internal/repository"
	serverservice "nullguard/internal/service/server"
	utils "nullguard/internal/service/wireguard"
)

func BuildClientObj(name, publicKey, privateKey, address, allowedIps, dnsServers, exposedLans string, fullTunnel bool,
	id, keepalive, serverId *int) domain.Client {

	var exposedLansPtr *string
	if exposedLans != "" {
		exposedLansPtr = &exposedLans
	}

	client := domain.Client{
		Name:        name,
		PublicKey:   publicKey,
		PrivateKey:  privateKey,
		AddressCidr: address,
		AllowedIps:  allowedIps,
		DnsServers:  dnsServers,
		FullTunnel:  fullTunnel,
		ExposedLans: exposedLansPtr,
	}

	// assign server id if provided
	if serverId != nil {
		client.ServerID = uint(*serverId)
	}

	// assign keepalive; use 30 if nil
	if keepalive != nil {
		client.Keepalive = *keepalive
	} else {
		client.Keepalive = 30 // default value
	}

	// assign client id if provided
	if id != nil && *id > 0 {
		client.ID = uint(*id)
	}

	return client
}

func ConvertRawToClient(rawData models.RawClientData) (domain.Client, error) {
	idPtr, err := httputil.ConvertJsonNumToIntPtr(rawData.ID)
	if err != nil {
		return domain.Client{}, err
	}

	serverID, err := httputil.ConvertJsonNumToIntPtr(rawData.ServerID)
	if err != nil {
		return domain.Client{}, err
	}

	keepalive, err := httputil.ConvertJsonNumToIntPtr(rawData.Keepalive)
	if err != nil || keepalive == nil {
		defaultKeepalive := 30 // default value if not provided or invalid
		keepalive = &defaultKeepalive
	}

	return BuildClientObj(rawData.Name, rawData.PublicKey, rawData.PrivateKey,
		rawData.AddressCidr, rawData.AllowedIps, rawData.DnsServers, rawData.ExposedLans, rawData.FullTunnel, idPtr, keepalive, serverID), nil
}

func Validate(client *domain.Client) error {
	client.Name = strings.TrimSpace(client.Name)
	client.AddressCidr = strings.TrimSpace(client.AddressCidr)
	client.PublicKey = strings.TrimSpace(client.PublicKey)
	client.PrivateKey = strings.TrimSpace(client.PrivateKey)

	if client.Name == "" {
		return fmt.Errorf("Client name cannot be empty")
	}

	if !validation.AllowedNameRegex.MatchString(client.Name) {
		return fmt.Errorf("Client name can only contain alphanumeric characters, dots (.), dashes (-), and underscores (_)")
	}

	server, err := serverservice.GetServerByID(int(client.ServerID))
	if err != nil {
		return err
	}

	if server.ID == 0 {
		return fmt.Errorf("Could not find an existing server with and id of %d", client.ServerID)
	}

	if client.PublicKey == "" {
		return fmt.Errorf("Client public key cannot be empty")
	}

	if client.PrivateKey == "" {
		return fmt.Errorf("Client private key cannot be empty")
	}

	if strings.Contains(client.PublicKey, " ") {
		return fmt.Errorf("Public key must not contain spaces")
	}

	if strings.Contains(client.PrivateKey, " ") {
		return fmt.Errorf("Private key must not contain spaces")
	}

	if client.AddressCidr == "" {
		return fmt.Errorf("Client address CIDR cannot be empty")
	}

	clientIp, err := validation.ValidateCIDR(client.AddressCidr)
	if err != nil {
		return err
	}

	if client.Keepalive < 0 {
		return fmt.Errorf("Keepalive cannot be negative")
	}
	if client.Keepalive > 600 {
		return fmt.Errorf("Keepalive cannot exceed 600 seconds")
	}

	// validate conflicts only after the address is already validated
	conflictingClients, err := repository.FindConflictingClients(*client)
	if err != nil {
		return fmt.Errorf("Error validating unique client properties: %v", err)
	}

	for _, conflictingClient := range conflictingClients {
		if conflictingClient.ID != client.ID {
			if conflictingClient.Name == client.Name {
				return fmt.Errorf("A client with the specified name already exists for this server: %s", conflictingClient.Name)
			}
			if conflictingClient.AddressCidr == client.AddressCidr {
				return fmt.Errorf("A client with the specified address CIDR already exists: %s", conflictingClient.AddressCidr)
			}
		}
	}

	serverIp, err := validation.ValidateCIDR(server.Address)
	if err != nil {
		return err
	}

	if strings.TrimSpace(clientIp) == strings.TrimSpace(serverIp) {
		return fmt.Errorf("Client cannot have the same address as the server")
	}

	if strings.TrimSpace(client.AllowedIps) == "" {
		return fmt.Errorf("Server CIDR cannot be empty")
	}

	allowedIpsCIDRs := strings.Split(client.AllowedIps, ",")
	for i := range allowedIpsCIDRs {
		if _, err := validation.ValidateCIDR(strings.TrimSpace(allowedIpsCIDRs[i])); err != nil {
			return err
		}
	}

	client.DnsServers = strings.TrimSpace(client.DnsServers)
	if client.DnsServers != "" {
		dnsEntries := strings.Split(client.DnsServers, ",")
		if len(dnsEntries) > 2 {
			return fmt.Errorf("Maximum of 2 DNS servers allowed")
		}
		for i := range dnsEntries {
			if err := validation.ValidateIP(strings.TrimSpace(dnsEntries[i])); err != nil {
				return err
			}
		}
	}

	if client.ExposedLans != nil {
		for _, c := range validation.SplitCidrCsv(*client.ExposedLans) {
			if _, err := validation.ValidateCIDR(c); err != nil {
				return err
			}
		}
	}

	return nil
}

func BuildDefaultClient() (domain.Client, error) {
	defaultClientName := config.GetEnv("WG_CLIENT_DEFAULT_NAME", "")
	privateKey, publicKey, err := utils.GenerateWgKeys("client")
	if err != nil {
		return domain.Client{}, fmt.Errorf("failed to generate client keys: %w", err)
	}

	return domain.Client{
		Name:       defaultClientName,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

func MapClientsToRawData(clients []domain.Client, peerStatuses map[string]serverservice.PeerStatus) []models.RawClientData {
	var rawClients []models.RawClientData

	for _, client := range clients {
		status := peerStatuses[client.PublicKey]
		rawClients = append(rawClients, models.RawClientData{
			ID:            json.Number(fmt.Sprintf("%d", client.ID)),
			Name:          client.Name,
			AddressCidr:   client.AddressCidr,
			AllowedIps:    client.AllowedIps,
			DnsServers:    client.DnsServers,
			Keepalive:     json.Number(fmt.Sprintf("%d", client.Keepalive)), // safe pointer dereference
			IsConnected:   status.IsConnected,
			LastHandshake: status.LastHandshake,
			TransferRx:    status.TransferRx,
			TransferTx:    status.TransferTx,
		})
	}

	return rawClients
}

func UpdateClient(oldClient *domain.Client, newClient domain.Client) error {
	database.UpdateFieldIfChanged(&oldClient.Name, newClient.Name)
	database.UpdateFieldIfChanged(&oldClient.AddressCidr, newClient.AddressCidr)
	database.UpdateFieldIfChanged(&oldClient.AllowedIps, newClient.AllowedIps)
	database.UpdateFieldIfChanged(&oldClient.DnsServers, newClient.DnsServers)
	database.UpdateFieldIfChanged(&oldClient.FullTunnel, newClient.FullTunnel)
	database.UpdateFieldIfChanged(&oldClient.Keepalive, newClient.Keepalive)
	database.UpdatePointerFieldIfChanged(&oldClient.ExposedLans, newClient.ExposedLans)

	// save the updated object via repository
	return repository.UpdateClient(oldClient)
}

func ClearClientSessionData(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.Store.Get(r, "update-client") // err safe to ignore: gorilla returns a valid empty session on decode failure
	delete(session.Values, "clientData")
	delete(session.Values, "serverInterface")
	delete(session.Values, "serverAddress")
	delete(session.Values, "serverSupernet")
	session.Save(r, w)
}

// GetClientByIDAndServerID retrieves a client by its ID and server ID
func GetClientByIDAndServerID(client domain.Client) (domain.Client, error) {
	return repository.FetchClientByIdAndServerId(client)
}

// GetServerClientsList retrieves all clients for a given server
func GetServerClientsList(server domain.Server) ([]domain.Client, error) {
	return repository.FetchServerClientsList(server)
}

// GetPeerExposedLans returns the deduped union of every client's ExposedLans on
// the given server, optionally excluding one client (pass 0 to include all).
// Use this to surface "reachable peer LANs" as a checkbox list on the client form.
func GetPeerExposedLans(server domain.Server, excludeClientID uint) ([]string, error) {
	clients, err := repository.FetchServerClientsList(server)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range clients {
		if c.ID == excludeClientID || c.ExposedLans == nil {
			continue
		}
		for _, cidr := range validation.SplitCidrCsv(*c.ExposedLans) {
			if !seen[cidr] {
				seen[cidr] = true
				out = append(out, cidr)
			}
		}
	}
	return out, nil
}

// CreateClient creates a new client in the database
func CreateClient(client domain.Client) error {
	return repository.CreateClient(client)
}

// DeleteClientByIDAndServerID deletes a client by its ID and server ID
func DeleteClientByIDAndServerID(clientID uint, serverID uint) error {
	return repository.DeleteClientByIDAndServerID(clientID, serverID)
}

// DeleteClientsByServerID deletes all clients for a given server
func DeleteClientsByServerID(serverID uint) error {
	return repository.DeleteClientsByServerID(serverID)
}

// GetAllClientsCount retrieves the total count of all clients
func GetAllClientsCount() (int64, error) {
	return repository.FetchAllClientsCount()
}
