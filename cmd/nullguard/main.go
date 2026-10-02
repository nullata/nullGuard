// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	router "nullguard/internal/api/http"
	"nullguard/internal/api/http/middleware"
	config "nullguard/internal/infrastructure/config"
	database "nullguard/internal/infrastructure/database"
	staticfileshandler "nullguard/internal/infrastructure/static"
	templatehandler "nullguard/internal/infrastructure/template"
	appversion "nullguard/internal/infrastructure/version"
	serverservice "nullguard/internal/service/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printBanner() {
	bannerBytes, err := os.ReadFile("banner.txt")
	if err == nil {
		fmt.Println(string(bannerBytes))
	}
	log.Printf("nullguard version %s (commit: %s, built: %s)", version, commit, date)
	log.Println()
}

func main() {
	printBanner()
	// load environment variables from .env file
	config.LoadEnv()

	// load the application version from the VERSION file
	appversion.ReadFrom("VERSION")

	// initialize session store (must be after LoadEnv)
	middleware.InitSessionStore()

	// get app configuration
	port := config.GetEnv("SERVER_PORT", "8080")

	// init the db
	database.InitDB()

	// auto-start servers if configured
	serverservice.AutoStartServers()

	// init base templates at startup
	templatehandler.InitializeBaseTemplates()

	r := router.SetupRouter()

	// serve static files
	staticfileshandler.ServeStaticFiles(r, "./static")

	// wrap the router with middleware
	securedRouter := middleware.SecurityHeaders(middleware.EnforceSecureCookies(r))

	// start server with the configured port
	log.Printf("Starting server on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, securedRouter))
}
