// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mainflux

import (
	"os"

	"github.com/nats-io/nats.go"
)

const (
	// DefNatsURL default NATS message broker URL
	DefNatsURL = nats.DefaultURL
)

// Env reads specified environment variable. If no value has been found,
// fallback is returned.
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
