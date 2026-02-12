// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"

	"nullguard/internal/domain"
	serverservice "nullguard/internal/service/server"

	"github.com/skip2/go-qrcode"
)

// GenerateClientConfig generates a WireGuard configuration file content for a client
func GenerateClientConfig(client domain.Client) (string, error) {
	// fetch the associated server
	server, err := serverservice.GetServerByID(int(client.ServerID))
	if err != nil {
		return "", fmt.Errorf("Failed to fetch server: %w", err)
	}

	// build the configuration
	var configBuilder strings.Builder

	// [Interface] section
	configBuilder.WriteString("[Interface]\n")
	configBuilder.WriteString(fmt.Sprintf("PrivateKey = %s\n", client.PrivateKey))
	configBuilder.WriteString(fmt.Sprintf("Address = %s\n", client.AddressCidr))

	// add DNS if configured on the client
	if client.DnsServers != "" {
		configBuilder.WriteString(fmt.Sprintf("DNS = %s\n", client.DnsServers))
	}

	configBuilder.WriteString("\n")

	// [Peer] section (server)
	configBuilder.WriteString("[Peer]\n")
	configBuilder.WriteString(fmt.Sprintf("PublicKey = %s\n", server.PublicKey))
	configBuilder.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", server.WANAddress, server.Port))
	configBuilder.WriteString(fmt.Sprintf("AllowedIPs = %s\n", client.AllowedIps))

	if client.Keepalive > 0 {
		configBuilder.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", client.Keepalive))
	}

	return configBuilder.String(), nil
}

// GenerateQRCode generates a QR code PNG image from the client configuration
func GenerateQRCode(client domain.Client) ([]byte, error) {
	config, err := GenerateClientConfig(client)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate config: %w", err)
	}

	// generate qr code at medium recovery level, 256x256 pixels
	qrCode, err := qrcode.Encode(config, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate QR code: %w", err)
	}

	return qrCode, nil
}

// CreateConfigZip creates a zip file containing the client configuration
func CreateConfigZip(client domain.Client) ([]byte, string, error) {
	config, err := GenerateClientConfig(client)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to generate config: %w", err)
	}

	// create a buffer to write the zip to
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// create the config file in the zip
	fileName := fmt.Sprintf("%s.conf", client.Name)
	fileWriter, err := zipWriter.Create(fileName)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to create zip file entry: %w", err)
	}

	// write the config content
	_, err = fileWriter.Write([]byte(config))
	if err != nil {
		return nil, "", fmt.Errorf("Failed to write config to zip: %w", err)
	}

	// close the zip writer
	err = zipWriter.Close()
	if err != nil {
		return nil, "", fmt.Errorf("Failed to close zip writer: %w", err)
	}

	zipFileName := fmt.Sprintf("%s-wireguard.zip", client.Name)
	return buf.Bytes(), zipFileName, nil
}
