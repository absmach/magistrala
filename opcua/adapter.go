// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package opcua

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/absmach/magistrala/opcua/db"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThing creates thingID:OPC-UA-nodeID route-map
	CreateThing(ctx context.Context, thingID, nodeID string) error

	// UpdateThing updates thingID:OPC-UA-nodeID route-map
	UpdateThing(ctx context.Context, thingID, nodeID string) error

	// RemoveThing removes thingID:OPC-UA-nodeID route-map
	RemoveThing(ctx context.Context, thingID string) error

	// CreateChannel creates channelID:OPC-UA-serverURI route-map
	CreateChannel(ctx context.Context, chanID, serverURI string) error

	// UpdateChannel updates channelID:OPC-UA-serverURI route-map
	UpdateChannel(ctx context.Context, chanID, serverURI string) error

	// RemoveChannel removes channelID:OPC-UA-serverURI route-map
	RemoveChannel(ctx context.Context, chanID string) error

	// ConnectThing creates thingID:channelID route-map
	ConnectThing(ctx context.Context, chanID, thingID string) error

	// DisconnectThing removes thingID:channelID route-map
	DisconnectThing(ctx context.Context, chanID, thingID string) error

	// Browse browses available nodes for a given OPC-UA Server URI and NodeID
	Browse(ctx context.Context, serverURI, namespace, identifier string) ([]BrowsedNode, error)
}

// Config OPC-UA Server.
type Config struct {
	ServerURI string
	NodeID    string
	Interval  string `env:"MG_OPCUA_ADAPTER_INTERVAL_MS"   envDefault:"1000"`
	Policy    string `env:"MG_OPCUA_ADAPTER_POLICY"        envDefault:""`
	Mode      string `env:"MG_OPCUA_ADAPTER_MODE"          envDefault:""`
	CertFile  string `env:"MG_OPCUA_ADAPTER_CERT_FILE"     envDefault:""`
	KeyFile   string `env:"MG_OPCUA_ADAPTER_KEY_FILE"      envDefault:""`
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	subscriber Subscriber
	browser    Browser
	thingsRM   RouteMapRepository
	channelsRM RouteMapRepository
	connectRM  RouteMapRepository
	cfg        Config
	logger     *slog.Logger
}

// New instantiates the OPC-UA adapter implementation.
func New(sub Subscriber, brow Browser, thingsRM, channelsRM, connectRM RouteMapRepository, cfg Config, log *slog.Logger) Service {
	return &adapterService{
		subscriber: sub,
		browser:    brow,
		thingsRM:   thingsRM,
		channelsRM: channelsRM,
		connectRM:  connectRM,
		cfg:        cfg,
		logger:     log,
	}
}

func (as *adapterService) CreateThing(ctx context.Context, thingID, nodeID string) error {
	return as.thingsRM.Save(ctx, thingID, nodeID)
}

func (as *adapterService) UpdateThing(ctx context.Context, thingID, nodeID string) error {
	return as.thingsRM.Save(ctx, thingID, nodeID)
}

func (as *adapterService) RemoveThing(ctx context.Context, thingID string) error {
	return as.thingsRM.Remove(ctx, thingID)
}

func (as *adapterService) CreateChannel(ctx context.Context, chanID, serverURI string) error {
	return as.channelsRM.Save(ctx, chanID, serverURI)
}

func (as *adapterService) UpdateChannel(ctx context.Context, chanID, serverURI string) error {
	return as.channelsRM.Save(ctx, chanID, serverURI)
}

func (as *adapterService) RemoveChannel(ctx context.Context, chanID string) error {
	return as.channelsRM.Remove(ctx, chanID)
}

func (as *adapterService) ConnectThing(ctx context.Context, chanID, thingID string) error {
	serverURI, err := as.channelsRM.Get(ctx, chanID)
	if err != nil {
		return err
	}

	nodeID, err := as.thingsRM.Get(ctx, thingID)
	if err != nil {
		return err
	}

	as.cfg.NodeID = nodeID
	as.cfg.ServerURI = serverURI

	c := fmt.Sprintf("%s:%s", chanID, thingID)
	if err := as.connectRM.Save(ctx, c, c); err != nil {
		return err
	}

	go func() {
		if err := as.subscriber.Subscribe(ctx, as.cfg); err != nil {
			as.logger.Warn(fmt.Sprintf("subscription failed: %s", err))
		}
	}()

	// Store subscription details
	return db.Save(serverURI, nodeID)
}

func (as *adapterService) Browse(ctx context.Context, serverURI, namespace, identifier string) ([]BrowsedNode, error) {
	nodeID := fmt.Sprintf("%s;%s", namespace, identifier)

	nodes, err := as.browser.Browse(serverURI, nodeID)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (as *adapterService) DisconnectThing(ctx context.Context, chanID, thingID string) error {
	c := fmt.Sprintf("%s:%s", chanID, thingID)
	return as.connectRM.Remove(ctx, c)
}
