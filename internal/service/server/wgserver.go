// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"fmt"
	"log"
	"strings"

	"nullguard/internal/api/http/models"
	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/config"
	database "nullguard/internal/infrastructure/database"
	"nullguard/internal/pkg/httputil"
	"nullguard/internal/pkg/validation"
	"nullguard/internal/repository"
	utils "nullguard/internal/service/wireguard"
)

// GetServerList retrieves a basic list of servers
func GetServerList() ([]models.ServerBasic, error) {
	return repository.FetchServerBaseList()
}

// GetServerByID retrieves a server by its ID
func GetServerByID(id int) (domain.Server, error) {
	return repository.FetchServerByID(id)
}

// GetFullServerList retrieves the complete list of servers
func GetFullServerList() ([]domain.Server, error) {
	return repository.FetchServerList()
}

// GetServerByIDAndKeys retrieves a server by ID and validates keys
func GetServerByIDAndKeys(id int, publicKey, privateKey string) (domain.Server, error) {
	return repository.FindServerByIdAndKeys(id, publicKey, privateKey)
}

// GetServerByIDAndInterfaceName retrieves a server by ID and validates interface name
func GetServerByIDAndInterfaceName(id int, interfaceName string) (domain.Server, error) {
	return repository.FindServerByIdAndInterfaceName(id, interfaceName)
}

// CreateServer creates a new server in the database
func CreateServer(server domain.Server) error {
	return repository.CreateServer(server)
}

func buildServerObj(interfaceName, comment, address, publicKey, privateKey,
	postUp, postDown, wanAddress, supernetCidr, bridgeNetworks string, id *int, port int, defaultKeepAlive *int, autoRestart bool) domain.Server {

	// pointers for optional fields like PostUp, PostDown, SupernetCIDR, and BridgeNetworks
	var postUpPtr, postDownPtr, supernetCidrPtr, bridgeNetworksPtr *string
	if postUp != "" {
		postUpPtr = &postUp
	}
	if postDown != "" {
		postDownPtr = &postDown
	}
	if supernetCidr != "" {
		supernetCidrPtr = &supernetCidr
	}
	if bridgeNetworks != "" {
		bridgeNetworksPtr = &bridgeNetworks
	}

	server := domain.Server{
		InterfaceName:    interfaceName,
		Comment:          comment,
		Address:          address,
		Port:             port,
		PublicKey:        publicKey,
		PrivateKey:       privateKey,
		PostUp:           postUpPtr,
		PostDown:         postDownPtr,
		WANAddress:       wanAddress,
		SupernetCidr:     supernetCidrPtr,
		BridgeNetworks:   bridgeNetworksPtr,
		DefaultKeepalive: defaultKeepAlive, // use a pointer for DefaultKeepAlive if its not zero (or if zero is valid - just assign directly)
		AutoRestart:      autoRestart,
	}

	// if the ID is provided and valid, set it; otherwise, let the DB generate it
	if id != nil && *id > 0 {
		server.ID = uint(*id)
	}

	return server
}

// converts RawServerData to a Server struct
func ConvertRawToServer(rawData models.RawServerData) (domain.Server, error) {
	// convert Port from json.Number to int
	var port int64 = 0
	if rawData.Port != "" {
		var err error
		port, err = rawData.Port.Int64()
		if err != nil {
			log.Printf("Error converting port: %v", err)
			return domain.Server{}, err
		}
	}

	// convert DefaultKeepAlive from json.Number to int
	keepAlive, err := httputil.ConvertJsonNumToIntPtr(rawData.DefaultKeepAlive)
	if err != nil {
		return domain.Server{}, err
	}

	// check if rawData.ID is empty before trying to convert it
	idPtr, err := httputil.ConvertJsonNumToIntPtr(rawData.ID)
	if err != nil {
		return domain.Server{}, err
	}

	return buildServerObj(rawData.InterfaceName, rawData.Comment, rawData.Address,
		rawData.PublicKey, rawData.PrivateKey, rawData.PostUp, rawData.PostDown,
		rawData.WANAddress, rawData.SupernetCIDR, rawData.BridgeNetworks, idPtr, int(port), keepAlive, rawData.AutoRestart), nil
}

func BuildNewServerConf(port int, address string) (domain.Server, error) {
	defaultName := "my-wg-server"
	defaultComment := ""

	defaultSupernetCidr := config.GetEnv("WG_SERVER_DEFAULT_SUPERNET", "")
	defaultKeepAlive := config.GetEnvInt("WG_SERVER_DEFAULT_KEEPALIVE", "")

	defaultHostNetworkInterface, err := getDefaultInterface()
	if err != nil {
		return domain.Server{}, fmt.Errorf("failed to get default interface: %w", err)
	}
	log.Printf("Create new server - Default network interface detected: %s", defaultHostNetworkInterface)

	defaultPostUp := fmt.Sprintf(postUpTemplate, defaultName, defaultName, defaultHostNetworkInterface)
	defaultPostDown := fmt.Sprintf(postDownTemplate, defaultName, defaultName, defaultHostNetworkInterface)

	wanAddress, err := getWANAddress()
	if err != nil {
		log.Printf("Error fetching WAN address: %v", err)
		wanAddress = "1.2.3.4" // set a w/e default value
	}

	privateKey, publicKey, err := utils.GenerateWgKeys("server")
	if err != nil {
		return domain.Server{}, fmt.Errorf("failed to generate server keys: %w", err)
	}

	return buildServerObj(defaultName, defaultComment, address, publicKey, privateKey, defaultPostUp,
		defaultPostDown, wanAddress, defaultSupernetCidr, "", nil, port, &defaultKeepAlive, false), nil
}

func BuildDefaultServerConf() (domain.Server, error) {
	// setup defaults
	defaultName := config.GetEnv("WG_SERVER_DEFAULT_NAME", "")
	defaultComment := config.GetEnv("WG_SERVER_DEFAULT_COMMENT", "")
	defaultAddress := config.GetEnv("WG_SERVER_DEFAULT_ADDR", "")
	defaultPort := config.GetEnvInt("WG_SERVER_DEFAULT_PORT", "")
	defaultSupernetCidr := config.GetEnv("WG_SERVER_DEFAULT_SUPERNET", "")
	defaultKeepAlive := config.GetEnvInt("WG_SERVER_DEFAULT_KEEPALIVE", "")

	defaultHostNetworkInterface, err := getDefaultInterface()
	if err != nil {
		return domain.Server{}, fmt.Errorf("failed to get default interface: %w", err)
	}
	log.Printf("Create new server - Default network interface detected: %s", defaultHostNetworkInterface)

	defaultPostUp := fmt.Sprintf(postUpTemplate, defaultName, defaultName, defaultHostNetworkInterface)
	defaultPostDown := fmt.Sprintf(postDownTemplate, defaultName, defaultName, defaultHostNetworkInterface)

	wanAddress, err := getWANAddress()
	if err != nil {
		log.Printf("Error fetching WAN address: %v", err)
		wanAddress = "1.2.3.4" // set a w/e default value
	}

	privateKey, publicKey, err := utils.GenerateWgKeys("server")
	if err != nil {
		return domain.Server{}, fmt.Errorf("failed to generate server keys: %w", err)
	}

	return buildServerObj(defaultName, defaultComment, defaultAddress, publicKey, privateKey, defaultPostUp,
		defaultPostDown, wanAddress, defaultSupernetCidr, "", nil, defaultPort, &defaultKeepAlive, false), nil
}

func Validate(server *domain.Server) error {
	server.InterfaceName = strings.TrimSpace(server.InterfaceName)
	server.PrivateKey = strings.TrimSpace(server.PrivateKey)
	server.PublicKey = strings.TrimSpace(server.PublicKey)
	server.Address = strings.TrimSpace(server.Address)
	server.WANAddress = strings.TrimSpace(server.WANAddress)

	if server.InterfaceName == "" {
		return fmt.Errorf("Interface Name cannot be empty")
	}

	if !validation.AllowedNameRegex.MatchString(server.InterfaceName) {
		return fmt.Errorf("Server name can only contain alphanumeric characters, dots (.), dashes (-), and underscores (_)")
	}

	if server.PrivateKey == "" || server.PublicKey == "" {
		return fmt.Errorf("Public and Private keys cannot be empty")
	}

	if strings.Contains(server.PublicKey, " ") {
		return fmt.Errorf("Public key must not contain spaces")
	}

	if strings.Contains(server.PrivateKey, " ") {
		return fmt.Errorf("Private key must not contain spaces")
	}

	if server.Address == "" || server.WANAddress == "" {
		return fmt.Errorf("Address and WAN Address cannot be empty")
	}

	if err := validation.ValidateIP(server.WANAddress); err != nil {
		return err
	}

	if _, err := validation.ValidateCIDR(server.Address); err != nil {
		return err
	}

	if server.Port == 0 {
		return fmt.Errorf("Server port set to %d. A valid server port is required", server.Port)
	}

	if server.SupernetCidr != nil && *server.SupernetCidr != "" {
		superNetCIDRs := strings.Split(*server.SupernetCidr, ",")
		for i := range superNetCIDRs {
			if _, err := validation.ValidateCIDR(strings.TrimSpace(superNetCIDRs[i])); err != nil {
				return err
			}
		}
	}

	if server.BridgeNetworks != nil && *server.BridgeNetworks != "" {
		bridgeCIDRs := strings.Split(*server.BridgeNetworks, ",")
		for i := range bridgeCIDRs {
			if _, err := validation.ValidateCIDR(strings.TrimSpace(bridgeCIDRs[i])); err != nil {
				return err
			}
		}
	}

	if server.DefaultKeepalive != nil {
		if *server.DefaultKeepalive < 0 {
			return fmt.Errorf("Default keepalive cannot be negative")
		}
		if *server.DefaultKeepalive > 600 {
			return fmt.Errorf("Default keepalive cannot exceed 600 seconds")
		}
	}

	conflictingServers, err := repository.FindConflictingServers(*server)
	if err != nil {
		return fmt.Errorf("Error validating unique server properties: %v", err)
	}

	for _, conflictingServer := range conflictingServers {
		if conflictingServer.ID != server.ID {
			if conflictingServer.Address == server.Address {
				return fmt.Errorf("A server with the specified address already exists: %s", conflictingServer.InterfaceName)
			}
			if conflictingServer.InterfaceName == server.InterfaceName {
				return fmt.Errorf("A server with the specified interface name already exists: %s", conflictingServer.InterfaceName)
			}
			if conflictingServer.Port == server.Port {
				return fmt.Errorf("A server with the specified port already exists: %s", conflictingServer.InterfaceName)
			}
		}
	}

	return nil
}

func ValidateDeployData(deployData models.DeployData) (*int, error) {
	if strings.TrimSpace(deployData.InterfaceName) == "" {
		return nil, fmt.Errorf("Interface name cannot be empty")
	}

	var idPtr *int
	if deployData.ID.String() != "" {
		// convert ID to an int64
		id, err := deployData.ID.Int64()
		if err != nil {
			log.Printf("Error converting id: %v", err)
			return nil, fmt.Errorf("Invalid deployment request Id")
		}

		// convert id to *int if its valid
		if id > 0 {
			idValue := int(id)
			idPtr = &idValue
		}
	}
	return idPtr, nil
}

func UpdateServer(oldServer *domain.Server, newServer domain.Server) error {
	// update fields using the helper function
	database.UpdateFieldIfChanged(&oldServer.InterfaceName, newServer.InterfaceName)
	database.UpdateFieldIfChanged(&oldServer.Comment, newServer.Comment)
	database.UpdateFieldIfChanged(&oldServer.Address, newServer.Address)
	database.UpdateFieldIfChanged(&oldServer.Port, newServer.Port)

	database.UpdatePointerFieldIfChanged(&oldServer.PostUp, newServer.PostUp)
	database.UpdatePointerFieldIfChanged(&oldServer.PostDown, newServer.PostDown)
	database.UpdateFieldIfChanged(&oldServer.WANAddress, newServer.WANAddress)
	database.UpdatePointerFieldIfChanged(&oldServer.SupernetCidr, newServer.SupernetCidr)
	database.UpdatePointerFieldIfChanged(&oldServer.BridgeNetworks, newServer.BridgeNetworks)
	database.UpdatePointerFieldIfChanged(&oldServer.DefaultKeepalive, newServer.DefaultKeepalive)
	database.UpdateFieldIfChanged(&oldServer.AutoRestart, newServer.AutoRestart)

	// save the updated object via repository
	return repository.UpdateServer(oldServer)
}

func DeleteServer(server *domain.Server) error {
	return repository.DeleteServer(server)
}
