// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/gorilla/websocket"
	"github.com/mainflux/mainflux"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/ws"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol            = "ws"
	readwriteBufferSize = 1024
)

var (
	errUnauthorizedAccess = errors.New("missing or invalid credentials provided")
	errMalformedSubtopic  = errors.New("malformed subtopic")
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  readwriteBufferSize,
		WriteBufferSize: readwriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	logger mflog.Logger
)

// MakeHandler returns http handler with handshake endpoint.
func MakeHandler(ctx context.Context, svc ws.Service, l mflog.Logger, instanceID string) http.Handler {
	logger = l

	mux := bone.New()
	mux.GetFunc("/channels/:chanID/messages", handshake(ctx, svc))
	mux.GetFunc("/channels/:chanID/messages/*", handshake(ctx, svc))
	mux.GetFunc("/version", mainflux.Health(protocol, instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
