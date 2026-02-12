// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package middleware

import (
	"crypto/rand"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"nullguard/internal/domain"
	"nullguard/internal/infrastructure/config"

	"github.com/gorilla/sessions"
)

var Store *sessions.CookieStore
var sessionMaxAge int

func init() {
	registerModelsForGobSerialization()
}

// InitSessionStore initializes the session store with the secret key
// Must be called after environment variables are loaded
func InitSessionStore() {
	secret := config.GetEnv("SESSION_SECRET_KEY", "")
	if secret == "" {
		log.Println("Warning: SESSION_SECRET_KEY not set, generating random key")
		log.Println("This means sessions will not persist across server restarts")
		secret = string(generateRandomKey(32))
	}

	cookieSecure := strings.ToLower(config.GetEnv("COOKIE_SECURE", "false")) == "true"

	sessionMaxAge = 3600 // default: 1 hour
	if v, err := strconv.Atoi(config.GetEnv("SESSION_MAX_AGE", "")); err == nil && v > 0 {
		sessionMaxAge = v
	}

	Store = sessions.NewCookieStore([]byte(secret))
	Store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true, // prevents js access to cookies
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode, // prevents cross-site request cookie sending
		MaxAge:   sessionMaxAge,
	}
	log.Printf("Session store initialized (max age: %ds)", sessionMaxAge)
}

// SecurityHeaders adds basic security headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

var secureCookieOnce sync.Once

// EnforceSecureCookies upgrades cookie Secure flag to true when HTTPS is detected
// via X-Forwarded-Proto header. Once enabled, it is never downgraded back.
func EnforceSecureCookies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			secureCookieOnce.Do(func() {
				Store.Options.Secure = true
				log.Println("Detected HTTPS via X-Forwarded-Proto, enabling secure cookies")
			})
		}
		next.ServeHTTP(w, r)
	})
}

func generateRandomKey(length int) []byte {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		log.Fatalf("Failed to generate random key: %v", err)
	}
	return key
}

func registerModelsForGobSerialization() {
	// gob is used for encoding and decoding go values into binary format
	// registering the models with gob allows them to be serialized and stored in sessions
	// this is necessary because gorilla/sessions uses gob
	// this allows for storing complex objects in sessions without converting them to json; then seriazing;
	// then when using them deserializing and convert from json to object
	gob.Register(&domain.Client{})
	gob.Register(&domain.Admin{})
}

// CreateAuthSession creates an authentication session for an admin.
// It invalidates any pre-existing session first to prevent session fixation attacks.
func CreateAuthSession(w http.ResponseWriter, r *http.Request, admin *domain.Admin) error {
	// invalidate any existing session to prevent session fixation
	if oldSess, err := Store.Get(r, "auth"); err == nil {
		oldSess.Options.MaxAge = -1
		_ = oldSess.Save(r, w)
	}

	// create a fresh session
	sess, err := Store.New(r, "auth")
	if err != nil {
		log.Printf("Error creating new session: %v", err)
		return errors.New("failed to create session")
	}

	sess.Values["authenticated"] = true
	sess.Values["admin_id"] = admin.ID
	sess.Values["username"] = admin.Username

	if err := sess.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		return errors.New("failed to save session")
	}

	return nil
}

// GetSessionMaxAge returns the configured session max age in seconds
func GetSessionMaxAge() int {
	return sessionMaxAge
}

// DestroyAuthSession destroys the authentication session
func DestroyAuthSession(w http.ResponseWriter, r *http.Request) error {
	sess, err := Store.Get(r, "auth")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		return errors.New("failed to get session")
	}

	sess.Values["authenticated"] = false
	delete(sess.Values, "admin_id")
	delete(sess.Values, "username")
	sess.Options.MaxAge = -1

	if err := sess.Save(r, w); err != nil {
		log.Printf("Error destroying session: %v", err)
		return errors.New("failed to destroy session")
	}

	return nil
}
