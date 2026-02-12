// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package httputil

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"nullguard/internal/api/http/models"
	"nullguard/internal/pkg/constants"
)

const maxRequestBodySize = 1 << 20 // 1 MB

func DecodeJsonObject(object interface{}, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	if err := json.NewDecoder(r.Body).Decode(object); err != nil {
		return err
	}
	return nil
}

// helper function to create and send a JSON response
func SendJSONResponse(w http.ResponseWriter, httpStatus int, status constants.Status, message string, payload interface{}) {
	response := models.StandardResponse{
		Timestamp: time.Now().Format(time.RFC3339),
		Status:    status,
		Message:   message,
		Data:      payload,
	}

	// set content type and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	// encode the response to json and write it to the response writer
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// cant change status/headers after writeHeader, just log
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func ConvertJsonNumToIntPtr(value json.Number) (*int, error) {
	// check if the value is empty or invalid
	if value == "" {
		return nil, nil // no value provided
	}

	// attempt to parse the number
	num, err := value.Int64()
	if err != nil {
		log.Printf("Error converting value to int64: %v", err)
		return nil, err
	}

	// always return a pointer to the number (including zero)
	numValue := int(num)
	return &numValue, nil
}

// SanitizeFilename removes characters unsafe for HTTP Content-Disposition  headers
func SanitizeFilename(name string) string {
	replacer := strings.NewReplacer(`"`, "", `\`, "", "\r", "", "\n", "")
	return replacer.Replace(name)
}

// ValidateAndDecodeJSON validates the HTTP method and decodes the request body into the specified type.
// Returns nil and false if validation fails (error response already sent to client).
func ValidateAndDecodeJSON[T any](w http.ResponseWriter, r *http.Request, expectedMethod string) (*T, bool) {
	if r.Method != expectedMethod {
		SendJSONResponse(w, http.StatusMethodNotAllowed, constants.StatusError, "Method not allowed", nil)
		return nil, false
	}

	var data T
	if err := DecodeJsonObject(&data, w, r); err != nil {
		log.Printf("Error decoding JSON in ValidateAndDecodeJSON: %v", err)
		SendJSONResponse(w, http.StatusBadRequest, constants.StatusError, "Invalid request body", nil)
		return nil, false
	}

	return &data, true
}

// GetClientIP extracts the client IP address from the request
func GetClientIP(r *http.Request) string {
	// try x-forwarded-for header first (for proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// x-forwarded-for can contain multiple ips, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// try x-real-ip header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// fall back to remoteaddr
	ip := r.RemoteAddr
	// remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
