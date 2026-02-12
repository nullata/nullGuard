// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package repository

import (
	"fmt"
	"log"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
)

func FetchClientByIdAndServerId(reqClient domain.Client) (domain.Client, error) {
	var client domain.Client
	result := database.DB.Where("id = ? AND server_id = ?", reqClient.ID, reqClient.ServerID).First(&client)
	if result.Error != nil {
		log.Printf("Error finding client by id and server_id: %v", result.Error)
		return domain.Client{}, result.Error
	}
	return client, nil
}
func FetchServerClientsList(server domain.Server) ([]domain.Client, error) {
	var clientList []domain.Client
	result := database.DB.Where("server_id = ?", server.ID).Find(&clientList)
	if result.Error != nil {
		log.Printf("Error fetching clients: %v", result.Error)
		return []domain.Client{}, result.Error
	}
	return clientList, nil
}

func FetchAllClientsCount() (int64, error) {
	var clientCount int64
	result := database.DB.Model(&domain.Client{}).Count(&clientCount)
	if result.Error != nil {
		log.Printf("Error counting clients: %v", result.Error)
		return 0, result.Error
	}
	return clientCount, nil
}

func CreateClient(client domain.Client) error {
	result := database.DB.Save(&client)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func FindConflictingClients(client domain.Client) ([]domain.Client, error) {
	var clientList []domain.Client
	result := database.DB.Where(
		"(name = ? AND server_id = ?) OR (address_cidr = ? AND server_id = ?)",
		client.Name, client.ServerID, client.AddressCidr, client.ServerID,
	).Find(&clientList)
	if result.Error != nil {
		log.Printf("Error fetching conflicting clients: %v", result.Error)
		return nil, result.Error
	}
	return clientList, nil
}

func DeleteClientByIDAndServerID(clientID uint, serverID uint) error {
	var client domain.Client
	result := database.DB.Where("id = ? AND server_id = ?", clientID, serverID).Delete(&client)
	if result.Error != nil {
		return fmt.Errorf("Error deleting client with ID %d and ServerID %d: %v", clientID, serverID, result.Error)
	}

	// check if client was actually deleted
	if result.RowsAffected == 0 {
		return fmt.Errorf("No client found with ID %d and ServerID %d", clientID, serverID)
	}

	return nil
}

func DeleteClientsByServerID(serverID uint) error {
	result := database.DB.Where("server_id = ?", serverID).Delete(&domain.Client{})
	if result.Error != nil {
		return fmt.Errorf("Error deleting clients for ServerID %d: %v", serverID, result.Error)
	}

	return nil
}

func UpdateClient(client *domain.Client) error {
	result := database.DB.Save(client)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
