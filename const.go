/**
 * Copyright (c) Mainflux
 *
 * FluxMQ is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package main

import (
	"time"
)

const (
	// Version is the current version for the server.
	Version = "0.0.1"

	// DefaultPort is the default port for client connections.
	DefaultPort = 1833

	// DefaultHost defaults to all interfaces.
	DefaultHost = "0.0.0.0"

	// MaxPayloadSize is the maximum allowed payload size. Should be using
	// something different if > 1MB payloads are needed.
	MaxPayloadSize = (1024 * 1024)

	// MaxPendingSize is the maximum outbound size (in bytes) per client.
	MaxPendingSize = (10 * 1024 * 1024)

	// DefaultMaxConnections is the default maximum connections allowed.
	DefaultMaxConnections = (64 * 1024)

	// AcceptMinSleep is the minimum acceptable sleep times on temporary errors.
	AcceptMinSleep = 10 * time.Millisecond

	// AcceptMaxSleep is the maximum acceptable sleep times on temporary errors
	AcceptMaxSleep = 1 * time.Second

	// EmptyString is empty string
	EmptyString = ""
)
