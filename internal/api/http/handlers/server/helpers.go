// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package server

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-sql-driver/mysql"
	pgxpgconn "github.com/jackc/pgx/v5/pgconn"
	"nullguard/internal/api/http/models"
	"nullguard/internal/domain"
	"nullguard/internal/pkg/constants"
	"nullguard/internal/pkg/httputil"
	serverservice "nullguard/internal/service/server"
)

// isDuplicateEntryError reports whether a repository error is a unique-constraint
// violation, regardless of the active database backend (MySQL error 1062,
// PostgreSQL SQLSTATE 23505, or SQLite "UNIQUE constraint failed").
func isDuplicateEntryError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	var pgErr *pgxpgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// validateAndGetServer handles common validation logic for server operations.
// It validates the HTTP method, decodes the request body, validates deploy data,
// and retrieves the server. Returns nil and false if validation fails (error response already sent).
func validateAndGetServer(w http.ResponseWriter, r *http.Request) (*domain.Server, bool) {
	if r.Method != http.MethodPost {
		httputil.SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return nil, false
	}

	var rawData models.DeployData
	if err := httputil.DecodeJsonObject(&rawData, w, r); err != nil {
		log.Printf("Error decoding JSON in validateAndGetServer: %v", err)
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Error parsing JSON", nil)
		return nil, false
	}

	serverID, err := serverservice.ValidateDeployData(rawData)
	if err != nil {
		httputil.SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, err.Error(), nil)
		return nil, false
	}

	server, err := serverservice.GetServerByIDAndInterfaceName(*serverID, rawData.InterfaceName)
	if err != nil {
		log.Printf("Error fetching server by ID and interface name: %v", err)
		httputil.SendJSONResponse(w, http.StatusInternalServerError, constants.StatusError, "Could not find requested server", nil)
		return nil, false
	}

	return &server, true
}
