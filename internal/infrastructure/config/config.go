// Copyright (c) 2026 nullata
// SPDX-License-Identifier: Elastic-2.0
// License: https://www.elastic.co/licensing/elastic-license

package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file
// Should be called once at application startup
func LoadEnv() {
	// get current working directory for debugging
	cwd, _ := os.Getwd()
	log.Printf("Looking for .env file in: %s", cwd)

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: could not load .env file: %v", err)
		log.Printf("Using system environment variables only")
	} else {
		log.Printf("Successfully loaded .env file")
	}
}

// GetEnv retrieves an environment variable with a default fallback
func GetEnv(envVar, defaultVal string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultVal
	}
	return value
}

// GetEnvInt retrieves an environment variable as an integer with a default fallback
func GetEnvInt(envVar, defaultVal string) int {
	value := GetEnv(envVar, defaultVal)
	parseInt, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("could not parse %s value to int: %s", envVar, value)
	}
	return parseInt
}
