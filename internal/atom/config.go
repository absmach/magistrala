// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second
const defaultAdminUsername = "admin"

// Config controls Magistrala's optional Atom integration.
type Config struct {
	URL           string
	JWKSURL       string
	JWTIssuer     string
	JWTAudience   string
	Token         string
	AdminUsername string
	AdminSecret   string
	Timeout       time.Duration
	UserAgent     string
}

// LoadConfig reads Atom integration settings from environment variables.
func LoadConfig() Config {
	atomURL := strings.TrimRight(os.Getenv("ATOM_URL"), "/")
	return Config{
		URL:           atomURL,
		JWKSURL:       envString("ATOM_JWKS_URL", atomURL+"/.well-known/jwks.json"),
		JWTIssuer:     envString("ATOM_JWT_ISSUER", envString("ATOM_PUBLIC_URL", atomURL)),
		JWTAudience:   envString("ATOM_JWT_AUDIENCE", "magistrala"),
		Token:         envString("ATOM_SERVICE_TOKEN", os.Getenv("ATOM_ADMIN_TOKEN")),
		AdminUsername: envString("ATOM_SERVICE_USERNAME", envString("ATOM_ADMIN_USERNAME", defaultAdminUsername)),
		AdminSecret:   envString("ATOM_SERVICE_SECRET", os.Getenv("ATOM_ADMIN_SECRET")),
		Timeout:       envDuration("ATOM_TIMEOUT", defaultTimeout),
		UserAgent:     "magistrala-atom-integration",
	}
}

func envString(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err == nil {
		return d
	}
	seconds, err := strconv.Atoi(v)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}
