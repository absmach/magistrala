// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package opcua

import (
	"context"
	"errors"
	"fmt"

	"github.com/mainflux/mainflux"
)

const (
	protocol      = "opcua"
	thingSuffix   = "thing"
	channelSuffix = "channel"
)

var (
	// ErrMalformedIdentity indicates malformed identity received (e.g.
	// invalid namespace or ID).
	ErrMalformedIdentity = errors.New("malformed identity received")

	// ErrMalformedMessage indicates malformed OPC-UA message.
	ErrMalformedMessage = errors.New("malformed message received")

	// ErrNotFoundIdentifier indicates a non-existent route map for a Node Identifier.
	ErrNotFoundIdentifier = errors.New("route map not found for this node identifier")

	// ErrNotFoundNamespace indicates a non-existent route map for an Node Namespace.
	ErrNotFoundNamespace = errors.New("route map not found for this node namespace")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThing creates thing  mfx:opc & opc:mfx route-map
	CreateThing(string, string) error

	// UpdateThing updates thing mfx:opc & opc:mfx route-map
	UpdateThing(string, string) error

	// RemoveThing removes thing mfx:opc & opc:mfx route-map
	RemoveThing(string) error

	// CreateChannel creates channel mfx:opc & opc:mfx route-map
	CreateChannel(string, string) error

	// UpdateChannel updates mfx:opc & opc:mfx route-map
	UpdateChannel(string, string) error

	// RemoveChannel removes channel mfx:opc & opc:mfx route-map
	RemoveChannel(string) error

	// Publish forwards messages from the OPC-UA MQTT broker to Mainflux NATS broker
	Publish(context.Context, string, Message) error
}

// Config OPC-UA Server
type Config struct {
	ServerURI      string
	NodeNamespace  string
	NodeIdintifier string
	Policy         string
	Mode           string
	CertFile       string
	KeyFile        string
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher  mainflux.MessagePublisher
	thingsRM   RouteMapRepository
	channelsRM RouteMapRepository
}

// New instantiates the OPC-UA adapter implementation.
func New(pub mainflux.MessagePublisher, thingsRM, channelsRM RouteMapRepository) Service {
	return &adapterService{
		publisher:  pub,
		thingsRM:   thingsRM,
		channelsRM: channelsRM,
	}
}

// Publish forwards messages from OPC-UA MQTT broker to Mainflux NATS broker
func (as *adapterService) Publish(ctx context.Context, token string, m Message) error {
	// Get route map of OPC-UA Node Namespace
	channelID, err := as.channelsRM.Get(m.Namespace)
	if err != nil {
		return ErrNotFoundNamespace
	}

	// Get route map of OPC-UA Node Identifier
	thingID, err := as.thingsRM.Get(m.ID)
	if err != nil {
		return ErrNotFoundIdentifier
	}

	// Publish on Mainflux NATS broker
	SenML := fmt.Sprintf(`[{"n":"opcua","v":%f}]`, m.Data)
	payload := []byte(SenML)
	msg := mainflux.Message{
		Publisher:   thingID,
		Protocol:    protocol,
		ContentType: "Content-Type",
		Channel:     channelID,
		Payload:     payload,
	}

	return as.publisher.Publish(ctx, token, msg)
}

func (as *adapterService) CreateThing(mfxDevID string, opcID string) error {
	return as.thingsRM.Save(mfxDevID, opcID)
}

func (as *adapterService) UpdateThing(mfxDevID string, opcID string) error {
	return as.thingsRM.Save(mfxDevID, opcID)
}

func (as *adapterService) RemoveThing(mfxDevID string) error {
	return as.thingsRM.Remove(mfxDevID)
}

func (as *adapterService) CreateChannel(mfxChanID string, opcNamespace string) error {
	return as.channelsRM.Save(mfxChanID, opcNamespace)
}

func (as *adapterService) UpdateChannel(mfxChanID string, opcNamespace string) error {
	return as.channelsRM.Save(mfxChanID, opcNamespace)
}

func (as *adapterService) RemoveChannel(mfxChanID string) error {
	return as.channelsRM.Remove(mfxChanID)
}
