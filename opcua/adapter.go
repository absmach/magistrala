// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package opcua

import (
	"errors"
	"fmt"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua/db"
)

const protocol = "opcua"

var (
	// ErrMalformedEntity indicates malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThing creates thingID:OPC-UA-nodeID route-map
	CreateThing(thingID, nodeID string) error

	// UpdateThing updates thingID:OPC-UA-nodeID route-map
	UpdateThing(thingID, nodeID string) error

	// RemoveThing removes thingID:OPC-UA-nodeID route-map
	RemoveThing(thingID string) error

	// CreateChannel creates channelID:OPC-UA-serverURI route-map
	CreateChannel(chanID, serverURI string) error

	// UpdateChannel updates channelID:OPC-UA-serverURI route-map
	UpdateChannel(chanID, serverURI string) error

	// RemoveChannel removes channelID:OPC-UA-serverURI route-map
	RemoveChannel(chanID string) error

	// ConnectThing creates thingID:channelID route-map
	ConnectThing(chanID, thingID string) error

	// DisconnectThing removes thingID:channelID route-map
	DisconnectThing(chanID, thingID string) error

	// Browse browses available nodes for a given OPC-UA Server URI and NodeID
	Browse(serverURI, namespace, identifier string) ([]BrowsedNode, error)
}

// Config OPC-UA Server
type Config struct {
	ServerURI string
	NodeID    string
	Interval  string
	Policy    string
	Mode      string
	CertFile  string
	KeyFile   string
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	subscriber Subscriber
	browser    Browser
	thingsRM   RouteMapRepository
	channelsRM RouteMapRepository
	connectRM  RouteMapRepository
	cfg        Config
	logger     logger.Logger
}

// New instantiates the OPC-UA adapter implementation.
func New(sub Subscriber, brow Browser, thingsRM, channelsRM, connectRM RouteMapRepository, cfg Config, log logger.Logger) Service {
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

func (as *adapterService) CreateThing(thingID, nodeID string) error {
	return as.thingsRM.Save(thingID, nodeID)
}

func (as *adapterService) UpdateThing(thingID, nodeID string) error {
	return as.thingsRM.Save(thingID, nodeID)
}

func (as *adapterService) RemoveThing(thingID string) error {
	return as.thingsRM.Remove(thingID)
}

func (as *adapterService) CreateChannel(chanID, serverURI string) error {
	return as.channelsRM.Save(chanID, serverURI)
}

func (as *adapterService) UpdateChannel(chanID, serverURI string) error {
	return as.channelsRM.Save(chanID, serverURI)
}

func (as *adapterService) RemoveChannel(chanID string) error {
	return as.channelsRM.Remove(chanID)
}

func (as *adapterService) ConnectThing(chanID, thingID string) error {
	serverURI, err := as.channelsRM.Get(chanID)
	if err != nil {
		return err
	}

	nodeID, err := as.thingsRM.Get(thingID)
	if err != nil {
		return err
	}

	as.cfg.NodeID = nodeID
	as.cfg.ServerURI = serverURI

	c := fmt.Sprintf("%s:%s", chanID, thingID)
	if err := as.connectRM.Save(c, c); err != nil {
		return err
	}

	go func() {
		if err := as.subscriber.Subscribe(as.cfg); err != nil {
			as.logger.Warn(fmt.Sprintf("subscription failed: %s", err))
		}
	}()

	// Store subscription details
	return db.Save(serverURI, nodeID)
}

func (as *adapterService) Browse(serverURI, namespace, identifier string) ([]BrowsedNode, error) {
	nodeID := fmt.Sprintf("%s;%s", namespace, identifier)

	nodes, err := as.browser.Browse(serverURI, nodeID)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (as *adapterService) DisconnectThing(chanID, thingID string) error {
	c := fmt.Sprintf("%s:%s", chanID, thingID)
	return as.connectRM.Remove(c)
}
