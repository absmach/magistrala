// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/channels"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for Channels API endpoints.
func MakeHandler(svc channels.Service, mux *chi.Mux, logger *slog.Logger, instanceID string) *chi.Mux {

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Route("/channels", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			createChannelEndpoint(svc),
			decodeCreateChannelReq,
			api.EncodeResponse,
			opts...,
		), "create_channel").ServeHTTP)

		r.Post("/bulk", otelhttp.NewHandler(kithttp.NewServer(
			createChannelsEndpoint(svc),
			decodeCreateChannelsReq,
			api.EncodeResponse,
			opts...,
		), "create_channels").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			listChannelsEndpoint(svc),
			decodeListChannels,
			api.EncodeResponse,
			opts...,
		), "list_channels").ServeHTTP)

		r.Route("/{channelID}", func(r chi.Router) {

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				viewChannelEndpoint(svc),
				decodeViewChannel,
				api.EncodeResponse,
				opts...,
			), "view_channel").ServeHTTP)

			r.Put("/", otelhttp.NewHandler(kithttp.NewServer(
				updateChannelEndpoint(svc),
				decodeUpdateChannel,
				api.EncodeResponse,
				opts...,
			), "update_channel").ServeHTTP)

			r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
				deleteChannelEndpoint(svc),
				decodeDeleteChannelReq,
				api.EncodeResponse,
				opts...,
			), "delete_channel").ServeHTTP)

			r.Post("/enable", otelhttp.NewHandler(kithttp.NewServer(
				enableChannelEndpoint(svc),
				decodeChangeChannelStatus,
				api.EncodeResponse,
				opts...,
			), "enable_channel").ServeHTTP)

			r.Post("/disable", otelhttp.NewHandler(kithttp.NewServer(
				disableChannelEndpoint(svc),
				decodeChangeChannelStatus,
				api.EncodeResponse,
				opts...,
			), "disable_channel").ServeHTTP)

			r.Post("/things/{thingID}/connect", otelhttp.NewHandler(kithttp.NewServer(
				connectChannelThingEndpoint(svc),
				decodeConnectChannelThingRequest,
				api.EncodeResponse,
				opts...,
			), "connect_channel_thing").ServeHTTP)

			r.Post("/things/{thingID}/disconnect", otelhttp.NewHandler(kithttp.NewServer(
				disconnectChannelThingEndpoint(svc),
				decodeDisconnectChannelThingRequest,
				api.EncodeResponse,
				opts...,
			), "disconnect_channel_thing").ServeHTTP)
		})

	})

	// Connect channel and thing
	mux.Post("/connect", otelhttp.NewHandler(kithttp.NewServer(
		connectEndpoint(svc),
		decodeConnectRequest,
		api.EncodeResponse,
		opts...,
	), "connect").ServeHTTP)

	// Disconnect channel and thing
	mux.Post("/disconnect", otelhttp.NewHandler(kithttp.NewServer(
		disconnectEndpoint(svc),
		decodeDisconnectRequest,
		api.EncodeResponse,
		opts...,
	), "disconnect").ServeHTTP)

	mux.Get("/health", magistrala.Health("channels", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
