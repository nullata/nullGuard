// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"strconv"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/config"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/validation"
	"nullguard/internal/repository"
	utils "nullguard/internal/service/wireguard"
)

var postUpTemplate = `PostUp = iptables -A FORWARD -i %s -j ACCEPT
PostUp = iptables -A FORWARD -o %s -j ACCEPT
PostUp = iptables -t nat -A POSTROUTING -o %s -j MASQUERADE`

var postDownTemplate = `PostDown = iptables -D FORWARD -i %s -j ACCEPT
PostDown = iptables -D FORWARD -o %s -j ACCEPT
PostDown = iptables -t nat -D POSTROUTING -o %s -j MASQUERADE`

const (
	// availableRangeStart begins at 10.240.0.0/24 - high enough to avoid
	// conflicts with common private networks that clients might already use
	// (e.g., 10.0.0.0/8, 10.1.0.0/16, etc.)
	availableRangeStart = 240
)

// fetches the WAN IP address of the local system
func getWANAddress() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "", fmt.Errorf("failed to fetch WAN IP address: %w", err)
	}
	defer response.Body.Close()

	// read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// trim and return the ip address as a string
	return strings.TrimSpace(string(body)), nil
}

// gets the default network interface name based on common criteria
func getDefaultInterface() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// check if the interface is up and supports broadcast
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagBroadcast != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			// check if the interface has an IPv4 address
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					return iface.Name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("default interface not found")
}

func findNextAvailableServerCIDR(serverList []domain.Server) (string, error) {
	usedAddresses := make(map[string]bool)
	for _, server := range serverList {
		usedAddresses[server.Address] = true
	}

	// sort the list of used addresses by the second octet to handle CIDR ranges properly
	for i := availableRangeStart; i <= 255; i++ {
		newCIDR := fmt.Sprintf("10.%d.0.1/24", i)
		if !usedAddresses[newCIDR] {
			return newCIDR, nil
		}
	}

	// ff no available address is found, wrap to the lower range (10.139.x.x and below)
	for i := availableRangeStart - 1; i >= 0; i-- {
		newCIDR := fmt.Sprintf("10.%d.0.1/24", i)
		if !usedAddresses[newCIDR] {
			return newCIDR, nil
		}
	}

	return "", fmt.Errorf("no available CIDR address found")
}

func findNextAvailableServerPort(serverList []domain.Server) int {
	usedPorts := make(map[int]bool)
	for _, server := range serverList {
		usedPorts[server.Port] = true
	}

	// start searching from the highest port + 1
	portList := []int{}
	for port := range usedPorts {
		portList = append(portList, port)
	}
	sort.Ints(portList)
	startPort := portList[len(portList)-1] + 1

	for {
		if !usedPorts[startPort] {
			return startPort
		}
		startPort++
	}
}

// find the first available values for interface name, port, and address CIDR
func FindAvailableServerAttributes(serverList []domain.Server) (int, string, error) {
	// find available port number
	nextPort := findNextAvailableServerPort(serverList)

	// gind available address CIDR
	nextAddress, err := findNextAvailableServerCIDR(serverList)
	if err != nil {
		return 0, "", err
	}

	return nextPort, nextAddress, nil
}

// getNormalizedServerConfigPath retrieves and normalizes the server config path.
// Returns the path with a trailing slash, or an error if the path is invalid.
func getNormalizedServerConfigPath() (string, error) {
	serverConfigPath := config.GetEnv("WG_SERVER_CONF_PATH", "./")
	isDir, err := utils.IsDirectory(serverConfigPath)
	if err != nil || !isDir {
		return "", err
	}
	// ensure the path ends with a trailing slash
	if string(serverConfigPath[len(serverConfigPath)-1]) != "/" {
		serverConfigPath = serverConfigPath + "/"
	}
	return serverConfigPath, nil
}

func GenerateServerConfig(server domain.Server) error {
	serverConfigPath, err := getNormalizedServerConfigPath()
	if err != nil {
		return err
	}

	// determine the comment value
	var comment = server.Comment
	if len(strings.TrimSpace(comment)) == 0 {
		comment = server.InterfaceName
	}

	// build the server conf str
	postUp := ""
	if server.PostUp != nil {
		postUp = *server.PostUp + "\n"
	}
	postDown := ""
	if server.PostDown != nil {
		postDown = *server.PostDown + "\n"
	}

	config := fmt.Sprintf(`[Interface]
# %s
Address = %s
SaveConfig = true
%s%sListenPort = %d
PrivateKey = %s
`,
		comment,
		server.Address,
		postUp,
		postDown,
		server.Port,
		server.PrivateKey,
	)

	// add client peers
	clients, err := repository.FetchServerClientsList(server)
	if err != nil {
		return fmt.Errorf("failed to fetch clients for server %s: %w", server.InterfaceName, err)
	}
	for _, client := range clients {
		allowedIPs := client.AddressCidr
		if client.ExposedLans != nil {
			for _, c := range validation.SplitCidrCsv(*client.ExposedLans) {
				allowedIPs += ", " + c
			}
		}
		config += fmt.Sprintf("\n[Peer]\n# %s\nPublicKey = %s\nAllowedIPs = %s\n",
			client.Name, client.PublicKey, allowedIPs)
		if client.Keepalive > 0 {
			config += fmt.Sprintf("PersistentKeepalive = %d\n", client.Keepalive)
		}
	}

	// write the configuration to a file
	fullConfigPath := serverConfigPath + server.InterfaceName + constants.WgServerConfExt
	if err := utils.WriteToFile(fullConfigPath, config, 0600); err != nil {
		return err
	}

	return nil
}

// PeerStatus holds connection and traffic data for a single WireGuard peer.
type PeerStatus struct {
	IsConnected   bool
	LastHandshake int64 // unix seconds; 0 if no handshake has occurred
	TransferRx    int64 // bytes received by server (client's upload)
	TransferTx    int64 // bytes sent by server (client's download)
}

// GetPeerStatuses returns connection and traffic info for all peers on the interface.
// A peer is considered connected if its latest WireGuard handshake occurred
// within 3 minutes. WireGuard peers re-handshake roughly every 2 minutes
// during an active session, so a 3-minute window accounts for normal jitter.
// When a peer disconnects, there is no explicit event - the handshake simply
// ages past this threshold and the peer is no longer considered connected.
// Transfer counters are cumulative since the interface was brought up.
func GetPeerStatuses(interfaceName string) map[string]PeerStatus {
	cmd := exec.Command("wg", "show", interfaceName, "dump")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return map[string]PeerStatus{}
	}

	peers := make(map[string]PeerStatus)
	lines := strings.Split(out.String(), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}

		publicKey := fields[0]
		handshake, _ := strconv.ParseInt(fields[4], 10, 64)
		rx, _ := strconv.ParseInt(fields[5], 10, 64)
		tx, _ := strconv.ParseInt(fields[6], 10, 64)

		peers[publicKey] = PeerStatus{
			IsConnected:   handshake > 0 && time.Since(time.Unix(handshake, 0)) <= 3*time.Minute,
			LastHandshake: handshake,
			TransferRx:    rx,
			TransferTx:    tx,
		}
	}

	return peers
}

func IsServerActive(server domain.Server) (bool, error) {
	// `wg show interfaces` prints just a whitespace-separated list of live
	// WireGuard interface names, which lets us match exactly. Using
	// `wg show` + strings.Contains is fragile: peer comments, public keys,
	// or another interface whose name contains ours (e.g. "wg0" inside
	// "wg01") would produce false positives.
	cmd := exec.Command("wg", "show", "interfaces")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("Error running wg command: %v", err)
	}

	for _, iface := range strings.Fields(out.String()) {
		if iface == server.InterfaceName {
			return true, nil
		}
	}
	return false, nil
}

// deletes the server configuration file
func DeleteServerConf(server domain.Server) error {
	serverConfigPath, err := getNormalizedServerConfigPath()
	if err != nil {
		return err
	}
	// write the configuration to a file
	fullConfigPath := serverConfigPath + server.InterfaceName + constants.WgServerConfExt
	isDir, err := utils.IsDirectory(fullConfigPath)
	if isDir {
		return fmt.Errorf("Parsed server configuration path %s is a directory", fullConfigPath)
	} else if err != nil {
		return err
	}

	if utils.FileExists(fullConfigPath) {
		if err := utils.DeleteFile(fullConfigPath); err != nil {
			return err
		}
	}

	log.Printf("Server configuration file deleted: %s", fullConfigPath)

	return nil

}

func StopServer(server domain.Server) error {
	// wg-quick down needs the .conf file to tear the interface down cleanly
	// (it runs the PostDown hooks from it). If the file is missing - typically
	// because /etc/wireguard was wiped by a container recreation while the
	// kernel wg0 interface survived - regenerate it from the DB first so
	// wg-quick has something to read. If wg-quick still fails but the
	// interface exists, fall back to `ip link delete` so the operator isn't
	// stuck with a "running" server they can't stop.
	serverConfigPath, pathErr := getNormalizedServerConfigPath()
	if pathErr == nil {
		fullConfigPath := serverConfigPath + server.InterfaceName + constants.WgServerConfExt
		if !utils.FileExists(fullConfigPath) {
			log.Printf("Server config missing at %s; regenerating from DB before stop", fullConfigPath)
			if err := GenerateServerConfig(server); err != nil {
				log.Printf("Failed to regenerate config for %s before stop: %v", server.InterfaceName, err)
			}
		}
	}

	cmd := exec.Command("wg-quick", "down", server.InterfaceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		wgQuickErr := stderr.String()

		// Last-resort teardown: if the kernel interface is still up, drop it
		// with `ip link delete`. This skips the PostDown hooks (leftover
		// iptables rules will need to be cleaned separately), but it does
		// unblock stop/restart on a mismatched-state server.
		if active, checkErr := IsServerActive(server); checkErr == nil && active {
			log.Printf("wg-quick down failed for %s (%s); falling back to `ip link delete`",
				server.InterfaceName, strings.TrimSpace(wgQuickErr))
			ipCmd := exec.Command("ip", "link", "delete", server.InterfaceName)
			var ipStderr bytes.Buffer
			ipCmd.Stderr = &ipStderr
			if ipErr := ipCmd.Run(); ipErr != nil {
				return fmt.Errorf("failed to stop server %s: wg-quick: %s; ip link delete: %s",
					server.InterfaceName, strings.TrimSpace(wgQuickErr), strings.TrimSpace(ipStderr.String()))
			}
			log.Printf("Server stopped via ip link delete: %s", server.InterfaceName)
			return nil
		}

		return fmt.Errorf("failed to stop server %s: %s", server.InterfaceName, wgQuickErr)
	}

	log.Printf("Server stopped: %s", server.InterfaceName)
	return nil
}

func AutoStartServers() {
	if strings.ToLower(config.GetEnv("AUTO_START_SERVERS", "false")) != "true" {
		return
	}

	servers, err := GetFullServerList()
	if err != nil {
		log.Printf("Auto-start: failed to fetch servers: %v", err)
		return
	}

	if len(servers) == 0 {
		log.Println("Auto-start: no servers found")
		return
	}

	for _, server := range servers {
		if err := GenerateServerConfig(server); err != nil {
			log.Printf("Auto-start: failed to generate config for %s: %v", server.InterfaceName, err)
			continue
		}
		if err := StartServer(server); err != nil {
			log.Printf("Auto-start: failed to start %s: %v", server.InterfaceName, err)
			continue
		}
		log.Printf("Auto-start: started %s", server.InterfaceName)
	}
}

// SetAutoRestart updates the auto-restart flag for a server
func SetAutoRestart(serverID int, enabled bool) error {
	server, err := GetServerByID(serverID)
	if err != nil {
		return fmt.Errorf("failed to fetch server: %w", err)
	}
	server.AutoRestart = enabled
	return repository.UpdateServer(&server)
}

// AutoRestartIfEnabled checks if a server has auto-restart enabled and is active,
// and if so, restarts it. Errors are logged but not returned since this is a
// best-effort operation that should not block the caller.
func AutoRestartIfEnabled(serverID int) {
	server, err := GetServerByID(serverID)
	if err != nil {
		log.Printf("Auto-restart: failed to fetch server %d: %v", serverID, err)
		return
	}

	if !server.AutoRestart {
		return
	}

	isActive, err := IsServerActive(server)
	if err != nil || !isActive {
		return
	}

	if err := StopServer(server); err != nil {
		log.Printf("Auto-restart: failed to stop server %s: %v", server.InterfaceName, err)
		return
	}

	if err := GenerateServerConfig(server); err != nil {
		log.Printf("Auto-restart: failed to generate config for %s: %v", server.InterfaceName, err)
		return
	}

	if err := StartServer(server); err != nil {
		log.Printf("Auto-restart: failed to start server %s: %v", server.InterfaceName, err)
		return
	}

	log.Printf("Auto-restart: restarted %s", server.InterfaceName)
}

func StartServer(server domain.Server) error {
	serverConfigPath := config.GetEnv("WG_SERVER_CONF_PATH", "./")
	if string(serverConfigPath[len(serverConfigPath)-1]) != "/" {
		serverConfigPath = serverConfigPath + "/"
	}
	fullConfigPath := serverConfigPath + server.InterfaceName + constants.WgServerConfExt

	cmd := exec.Command("wg-quick", "up", fullConfigPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start server %s: %s", server.InterfaceName, stderr.String())
	}

	log.Printf("Server started: %s", server.InterfaceName)
	return nil
}
