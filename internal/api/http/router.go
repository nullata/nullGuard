// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package router

import (
	"net/http"
	"strings"

	appcontroller "nullguard/internal/api/http/handlers/app"
	authcontroller "nullguard/internal/api/http/handlers/auth"
	clientcontroller "nullguard/internal/api/http/handlers/client"
	helpcontroller "nullguard/internal/api/http/handlers/help"
	servercontroller "nullguard/internal/api/http/handlers/server"
	"nullguard/internal/api/http/middleware"
	"nullguard/internal/infrastructure/template"

	"github.com/gorilla/mux"
)

func SetupRouter() *mux.Router {
	router := mux.NewRouter()

	// custom error pages with auth check
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			http.NotFound(w, r)
			return
		}
		if !middleware.IsAuthenticated(r) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		template.ErrorPageHandler(w, "Page not found", http.StatusNotFound)
	})
	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !middleware.IsAuthenticated(r) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		template.ErrorPageHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	////////////////////////
	// unprotected routes
	////////////////////////

	// health check
	router.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	// auth
	router.HandleFunc("/setup", authcontroller.SetupPageHandler).Methods("GET")
	router.HandleFunc("/api/v1/setup", authcontroller.SetupHandler).Methods("POST")
	router.HandleFunc("/login", authcontroller.LoginPageHandler).Methods("GET")
	router.HandleFunc("/api/v1/login", authcontroller.LoginHandler).Methods("POST")
	router.HandleFunc("/api/v1/get-password-hint", authcontroller.GetPasswordHintHandler).Methods("GET")

	////////////////////////
	// protected routes (subrouter with auth middleware)
	////////////////////////
	protected := router.PathPrefix("/").Subrouter()
	protected.Use(middleware.ChainMiddleware(
		middleware.RequireSetup,
		middleware.RequireAuthFlexible,
	))

	// auth
	protected.HandleFunc("/api/v1/logout", authcontroller.LogoutHandler).Methods("POST")
	protected.HandleFunc("/api/v1/change-password", authcontroller.ChangePasswordHandler).Methods("POST")

	// admin settings page
	protected.HandleFunc("/admin/settings", authcontroller.AdminSettingsPageHandler).Methods("GET")

	// API token management (session auth only, not bearer)
	protected.HandleFunc("/api/v1/tokens", authcontroller.CreateApiTokenHandler).Methods("POST")
	protected.HandleFunc("/api/v1/tokens", authcontroller.ListApiTokensHandler).Methods("GET")
	protected.HandleFunc("/api/v1/tokens/{id}", authcontroller.RevokeApiTokenHandler).Methods("DELETE")

	// main app
	protected.HandleFunc("/", appcontroller.AppMainPageHandler).Methods("GET")

	// wg server
	protected.HandleFunc("/server", servercontroller.ServerPageHandler).Methods("GET")
	protected.HandleFunc("/create-server", servercontroller.CreateServerPageHandler).Methods("GET")

	protected.HandleFunc("/api/v1/list-servers", servercontroller.ListServers).Methods("GET")
	protected.HandleFunc("/api/v1/fetch-server", servercontroller.FetchServer).Methods("POST")
	protected.HandleFunc("/api/v1/create-server", servercontroller.CreateServer).Methods("POST")
	protected.HandleFunc("/api/v1/update-server", servercontroller.UpdateServer).Methods("PUT")
	protected.HandleFunc("/api/v1/delete-server", servercontroller.DeleteServer).Methods("DELETE")
	protected.HandleFunc("/api/v1/deploy-server", servercontroller.DeployServer).Methods("POST")
	protected.HandleFunc("/api/v1/stop-server", servercontroller.StopServer).Methods("POST")
	protected.HandleFunc("/api/v1/restart-server", servercontroller.RestartServer).Methods("POST")
	protected.HandleFunc("/api/v1/toggle-auto-restart", servercontroller.ToggleAutoRestart).Methods("POST")

	// wg client
	protected.HandleFunc("/client", clientcontroller.ClientPageHandler).Methods("GET")
	protected.HandleFunc("/create-client", clientcontroller.CreateClientPageHandler).Methods("GET")
	protected.HandleFunc("/update-client", clientcontroller.UpdateClientPageHandler).Methods("GET")

	protected.HandleFunc("/api/v1/list-clients/{serverId}", clientcontroller.ListClients).Methods("GET")
	protected.HandleFunc("/api/v1/create-client", clientcontroller.CreateClient).Methods("POST")
	protected.HandleFunc("/api/v1/build-client", clientcontroller.BuildClient).Methods("POST")
	protected.HandleFunc("/api/v1/load-clients", clientcontroller.LoadClients).Methods("POST")
	protected.HandleFunc("/api/v1/delete-client", clientcontroller.DeleteClient).Methods("DELETE")
	protected.HandleFunc("/api/v1/update-client", clientcontroller.UpdateClient).Methods("PUT")

	protected.HandleFunc("/api/v1/set-create-client-session", clientcontroller.SetCreateClientSessionHandler).Methods("POST")
	protected.HandleFunc("/api/v1/set-update-client-session", clientcontroller.SetClientSessionHandler).Methods("POST")
	protected.HandleFunc("/api/v1/clear-update-client-session", clientcontroller.ClearClientSessionHandler).Methods("DELETE")

	// in-app field cheatsheets surfaced from clickable tooltip icons
	protected.HandleFunc("/help/{key}", helpcontroller.CheatsheetHandler).Methods("GET")

	// client config download routes
	protected.HandleFunc("/api/v1/client/{serverId}/{clientId}/config", clientcontroller.GetClientConfig).Methods("GET")
	protected.HandleFunc("/api/v1/client/{serverId}/{clientId}/qrcode", clientcontroller.GetClientQRCode).Methods("GET")
	protected.HandleFunc("/api/v1/client/{serverId}/{clientId}/download", clientcontroller.DownloadClientConfig).Methods("GET")

	return router
}
