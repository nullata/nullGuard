// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package client

import (
	"fmt"
	"log"
	"net"
	"strings"

	"nullguard/internal/domain"
	"nullguard/internal/repository"
)

// takes an ip in cidr notation (eg "10.252.0.2/24") and returns the subnet base with cidr  suffix
func GetSubnetBaseCIDR(cidr string) (string, net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Printf("Error parsing CIDR: %v", err)
		return "", *ipNet, err
	}

	ones, _ := ipNet.Mask.Size() // get the prefix size (eg 24 for /24)

	// use the network ip address as the base and return with cidr notation
	return fmt.Sprintf("%s/%d", ipNet.IP.String(), ones), *ipNet, nil
}

// inds the next available ip address within the server subnet excluding the servers own ip and already used client ips, returning it as a /32 cidr
func FindNextAvailableClientCidrAddress(server domain.Server, serverSubnetCidr net.IPNet) (string, error) {
	// parse and normalize the server address && removing cidr suffix if present
	serverIP := net.ParseIP(strings.Split(server.Address, "/")[0])
	usedAddresses := make(map[string]bool)
	usedAddresses[serverIP.String()] = true

	// add existing client addresses to the map
	clients, err := repository.FetchServerClientsList(server)
	if err != nil {
		return "", err
	}
	for _, client := range clients {
		// get the ip by splitting the mask
		clientIP := net.ParseIP(strings.Split(client.AddressCidr, "/")[0])
		usedAddresses[clientIP.String()] = true
	}

	// helper function to increment ip address
	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	// compute broadcast address (network address | ~mask)
	broadcastIP := make(net.IP, len(serverSubnetCidr.IP))
	for i := range serverSubnetCidr.IP {
		broadcastIP[i] = serverSubnetCidr.IP[i] | ^serverSubnetCidr.Mask[i]
	}

	// iterate over the ip addresses within the subnet
	for ip := serverSubnetCidr.IP.Mask(serverSubnetCidr.Mask); serverSubnetCidr.Contains(ip); inc(ip) {
		ipStr := ip.String()
		if !usedAddresses[ipStr] && !ip.Equal(serverSubnetCidr.IP) && !ip.Equal(broadcastIP) {
			// return the first available ip as a cidr with /32 mask
			return fmt.Sprintf("%s/32", ipStr), nil
		}
	}

	// if no available address is found, return an error
	return "", fmt.Errorf("No available CIDR address found")
}
