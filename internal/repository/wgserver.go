// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package repository

import (
	"log"

	"nullguard/internal/api/http/models"
	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/database"
)

func FetchServerByID(id int) (domain.Server, error) {
	var server domain.Server
	result := database.DB.Where("id = ?", id).First(&server)
	if result.Error != nil {
		log.Printf("Error fetching server: %v", result.Error)
		return domain.Server{}, result.Error
	}
	return server, nil
}

func FetchServerBaseList() ([]models.ServerBasic, error) {
	var serverList []models.ServerBasic
	result := database.DB.Model(&domain.Server{}).Select("id, interface_name, address").Find(&serverList)
	if result.Error != nil {
		log.Printf("Error fetching servers: %v", result.Error)
		return []models.ServerBasic{}, result.Error
	}
	return serverList, nil
}

func FetchServerList() ([]domain.Server, error) {
	var serverList []domain.Server
	result := database.DB.Find(&serverList)
	if result.Error != nil {
		log.Printf("Error fetching servers: %v", result.Error)
		return []domain.Server{}, result.Error
	}
	return serverList, nil
}

func FindServerByIdAndKeys(id int, publicKey, privateKey string) (domain.Server, error) {
	var server domain.Server
	result := database.DB.Where("id = ? AND public_key = ? AND private_key = ?", id, publicKey, privateKey).First(&server)

	if result.Error != nil {
		log.Printf("Error fetching server: %v", result.Error)
		return domain.Server{}, result.Error
	}
	return server, nil
}

func FindServerByIdAndInterfaceName(id int, interfaceName string) (domain.Server, error) {
	var server domain.Server
	result := database.DB.Where("id = ? AND interface_name = ?", id, interfaceName).First(&server)

	if result.Error != nil {
		log.Printf("Error fetching server: %v", result.Error)
		return domain.Server{}, result.Error
	}
	return server, nil
}

func CreateServer(server domain.Server) error {
	result := database.DB.Save(&server)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func FindConflictingServers(server domain.Server) ([]domain.Server, error) {
	var serverList []domain.Server
	result := database.DB.Where("address = ? OR interface_name = ? OR port = ?", server.Address, server.InterfaceName, server.Port).Find(&serverList)
	if result.Error != nil {
		log.Printf("Error fetching conflicting servers: %v", result.Error)
		return nil, result.Error
	}
	return serverList, nil
}

func UpdateServer(server *domain.Server) error {
	result := database.DB.Save(server)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func DeleteServer(server *domain.Server) error {
	result := database.DB.Unscoped().Delete(server)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
