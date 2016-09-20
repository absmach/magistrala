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
	// VERSION is the current version for the server.
	VERSION = "0.0.1"

	// DEFAULT_PORT is the default port for client connections.
	DEFAULT_PORT = 1833

	// DEFAULT_HOST defaults to all interfaces.
	DEFAULT_HOST = "0.0.0.0"

	// MAX_PAYLOAD_SIZE is the maximum allowed payload size. Should be using
	// something different if > 1MB payloads are needed.
	MAX_PAYLOAD_SIZE = (1024 * 1024)

	// MAX_PENDING_SIZE is the maximum outbound size (in bytes) per client.
	MAX_PENDING_SIZE = (10 * 1024 * 1024)

	// DEFAULT_MAX_CONNECTIONS is the default maximum connections allowed.
	DEFAULT_MAX_CONNECTIONS = (64 * 1024)

	// ACCEPT_MIN_SLEEP is the minimum acceptable sleep times on temporary errors.
	ACCEPT_MIN_SLEEP = 10 * time.Millisecond

	// ACCEPT_MAX_SLEEP is the maximum acceptable sleep times on temporary errors
	ACCEPT_MAX_SLEEP = 1 * time.Second

	// _EMPTY_ is empty string
	_EMPTY_ = ""
)
