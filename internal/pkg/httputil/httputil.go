// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package httputil

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"nullguard/internal/api/http/models"
	"nullguard/internal/infrastructure/config"
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

// trustedProxies caches the parsed TRUSTED_PROXIES env value, re-parsing
// only when the raw env string changes (so tests using t.Setenv see the
// new value immediately).
var (
	trustedProxiesMu   sync.Mutex
	trustedProxiesRaw  string
	trustedProxiesInit bool
	trustedProxies     []*net.IPNet
)

// parseTrustedProxies parses TRUSTED_PROXIES: a comma-separated list of CIDR
// notations (a bare IP is accepted and treated as /32 or /128). Entries that
// fail to parse are logged and skipped. An empty/absent value means no proxy
// is ever trusted and forwarded headers are ignored entirely.
func parseTrustedProxies() []*net.IPNet {
	raw := config.GetEnv("TRUSTED_PROXIES", "")

	trustedProxiesMu.Lock()
	defer trustedProxiesMu.Unlock()
	if trustedProxiesInit && trustedProxiesRaw == raw {
		return trustedProxies
	}

	var nets []*net.IPNet
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			ip := net.ParseIP(entry)
			if ip == nil {
				log.Printf("TRUSTED_PROXIES: ignoring unparseable entry %q", entry)
				continue
			}
			if ip.To4() != nil {
				entry += "/32"
			} else {
				entry += "/128"
			}
		}
		_, ipnet, err := net.ParseCIDR(entry)
		if err != nil {
			log.Printf("TRUSTED_PROXIES: ignoring unparseable entry %q", entry)
			continue
		}
		nets = append(nets, ipnet)
	}

	trustedProxiesRaw = raw
	trustedProxiesInit = true
	trustedProxies = nets
	return nets
}

func isTrustedProxy(ip net.IP, proxies []*net.IPNet) bool {
	for _, p := range proxies {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// GetClientIP extracts the client IP address from the request.
//
// X-Forwarded-For / X-Real-IP are honored ONLY when the direct TCP peer
// (RemoteAddr) falls inside a CIDR from TRUSTED_PROXIES (comma-separated;
// default empty = never trust forwarded headers). Unconditionally trusting
// those headers let any direct client spoof the recorded audit IPs (#17).
//
// When the peer is trusted, X-Forwarded-For is walked right-to-left: each
// rightmost entry that is itself a trusted proxy is skipped, and the first
// remaining address is returned. This is the only position a client cannot
// forge past a well-behaved appending proxy (leftmost entries are
// client-controlled whenever a proxy forwards them verbatim).
//
// The RemoteAddr fallback uses net.SplitHostPort, which handles IPv6
// bracket notation ([::1]:1234 -> ::1) that a naive LastIndex(":") split
// mangled.
func GetClientIP(r *http.Request) string {
	// direct peer address, port stripped the IPv6-aware way
	peerHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr without a port (rare; e.g. some proxies/Unix) —
		// treat the whole string as the address
		peerHost = r.RemoteAddr
	}
	peerIP := net.ParseIP(strings.TrimSpace(peerHost))

	proxies := parseTrustedProxies()
	if peerIP == nil || len(proxies) == 0 || !isTrustedProxy(peerIP, proxies) {
		if peerIP != nil {
			return peerIP.String()
		}
		return strings.TrimSpace(peerHost)
	}

	// peer is a trusted proxy: X-Forwarded-For wins, walked right-to-left
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		entries := strings.Split(xff, ",")
		for i := len(entries) - 1; i >= 0; i-- {
			candidate := net.ParseIP(strings.TrimSpace(entries[i]))
			if candidate == nil {
				// unparseable entry in the middle: stop here; whatever we
				// have walked so far is untrustworthy
				break
			}
			if !isTrustedProxy(candidate, proxies) {
				return candidate.String()
			}
		}
		// every entry was a trusted proxy (or unparseable): fall through
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		if ip := net.ParseIP(strings.TrimSpace(realIP)); ip != nil {
			return ip.String()
		}
	}

	if peerIP != nil {
		return peerIP.String()
	}
	return strings.TrimSpace(peerHost)
}
