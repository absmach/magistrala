package lora

import (
	"encoding/base64"
	"errors"

	"github.com/mainflux/mainflux"
)

const (
	protocol      = "lora"
	thingSuffix   = "thing"
	channelSuffix = "channel"
)

var (
	// ErrMalformedIdentity indicates malformed identity received (e.g.
	// invalid appID or deviceEUI).
	ErrMalformedIdentity = errors.New("malformed identity received")

	// ErrMalformedMessage indicates malformed LoRa message.
	ErrMalformedMessage = errors.New("malformed message received")

	// ErrNotFoundDev indicates a non-existent route map for a device EUI.
	ErrNotFoundDev = errors.New("route map not found for this device EUI")

	// ErrNotFoundApp indicates a non-existent route map for an application ID.
	ErrNotFoundApp = errors.New("route map not found for this application ID")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThing creates thing  mfx:lora & lora:mfx route-map
	CreateThing(string, string) error

	// UpdateThing updates thing mfx:lora & lora:mfx route-map
	UpdateThing(string, string) error

	// RemoveThing removes thing mfx:lora & lora:mfx route-map
	RemoveThing(string) error

	// CreateChannel creates channel mfx:lora & lora:mfx route-map
	CreateChannel(string, string) error

	// UpdateChannel updates mfx:lora & lora:mfx route-map
	UpdateChannel(string, string) error

	// RemoveChannel removes channel mfx:lora & lora:mfx route-map
	RemoveChannel(string) error

	// Publish forwards messages from the LoRa MQTT broker to Mainflux NATS broker
	Publish(Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher  mainflux.MessagePublisher
	thingsRM   RouteMapRepository
	channelsRM RouteMapRepository
}

// New instantiates the LoRa adapter implementation.
func New(pub mainflux.MessagePublisher, thingsRM, channelsRM RouteMapRepository) Service {
	return &adapterService{
		publisher:  pub,
		thingsRM:   thingsRM,
		channelsRM: channelsRM,
	}
}

// Publish forwards messages from Lora MQTT broker to Mainflux NATS broker
func (as *adapterService) Publish(m Message) error {
	// Get route map of lora application
	thing, err := as.thingsRM.Get(m.DevEUI)
	if err != nil {
		return ErrNotFoundDev
	}

	// Get route map of lora application
	channel, err := as.channelsRM.Get(m.ApplicationID)
	if err != nil {
		return ErrNotFoundApp
	}

	payload, err := base64.StdEncoding.DecodeString(m.Data)
	if err != nil {
		return ErrMalformedMessage
	}

	// Publish on Mainflux NATS broker
	msg := mainflux.RawMessage{
		Publisher:   thing,
		Protocol:    protocol,
		ContentType: "Content-Type",
		Channel:     channel,
		Payload:     payload,
	}

	return as.publisher.Publish(msg)
}

func (as *adapterService) CreateThing(mfxDevID string, loraDevEUI string) error {
	return as.thingsRM.Save(mfxDevID, loraDevEUI)
}

func (as *adapterService) UpdateThing(mfxDevID string, loraDevEUI string) error {
	return as.thingsRM.Save(mfxDevID, loraDevEUI)
}

func (as *adapterService) RemoveThing(mfxDevID string) error {
	return as.thingsRM.Remove(mfxDevID)
}

func (as *adapterService) CreateChannel(mfxChanID string, loraAppID string) error {
	return as.channelsRM.Save(mfxChanID, loraAppID)
}

func (as *adapterService) UpdateChannel(mfxChanID string, loraAppID string) error {
	return as.channelsRM.Save(mfxChanID, loraAppID)
}

func (as *adapterService) RemoveChannel(mfxChanID string) error {
	return as.channelsRM.Remove(mfxChanID)
}
