// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package static

import (
	"net/http"

	"github.com/gorilla/mux"
)

// sets up the static file handler for the router
// it takes a *mux.Router and a base path for serving the files
func ServeStaticFiles(router *mux.Router, basePath string) {
	staticDir := http.Dir(basePath)
	staticFileHandler := http.StripPrefix("/static/", http.FileServer(staticDir))
	router.PathPrefix("/static/").Handler(staticFileHandler).Methods("GET")
}
