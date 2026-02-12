// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package template

import (
	"net/http"
)

func ErrorPageHandler(w http.ResponseWriter, errMessage string, statusCode int) {
	// check if headers have already been written to prevent superfluous calls
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/html")
	}
	// clear the response if no content was already sent
	w.WriteHeader(statusCode)

	data := map[string]interface{}{
		"Title":        "nullguard - Oops!",
		"ErrorMessage": errMessage,
	}

	TemplateHandler(w, "templates/error.html", data)
}
