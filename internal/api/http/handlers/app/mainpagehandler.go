// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package app

import (
	"net/http"
)

func AppMainPageHandler(w http.ResponseWriter, r *http.Request) {
	// redirect to the server page after login
	http.Redirect(w, r, "/server", http.StatusSeeOther)
}
