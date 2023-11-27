// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/absmach/magistrala"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/ws"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	service             = "ws"
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
	logger mglog.Logger
)

// MakeHandler returns http handler with handshake endpoint.
func MakeHandler(ctx context.Context, svc ws.Service, l mglog.Logger, instanceID string) http.Handler {
	logger = l

	mux := chi.NewRouter()
	mux.Get("/channels/{chanID}/messages", handshake(ctx, svc))
	mux.Get("/channels/{chanID}/messages/*", handshake(ctx, svc))

	mux.Get("/health", magistrala.Health(service, instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
