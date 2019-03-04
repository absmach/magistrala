//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/mainflux/mainflux"
	mfsdk "github.com/mainflux/mainflux/sdk/go"
)

const (
	thingType = "device"
	chanName  = "channel"
)

var (
	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrMalformedEntity indicates malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrConflict indicates that entity with the same ID or external ID already exists.
	ErrConflict = errors.New("entity already exists")

	// ErrThings indicates failure to communicate with Mainflux Things service.
	// It can be due to networking error or invalid/unauthorized request.
	ErrThings = errors.New("error receiving response from Things service")
)

var _ Service = (*bootstrapService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Add adds new Thing Config to the user identified by the provided key.
	Add(string, Config) (Config, error)

	// View returns Thing Config with given ID belonging to the user identified by the given key.
	View(string, string) (Config, error)

	// Update updates editable fields of the provided Config.
	Update(string, Config) error

	// UpdateConnections updates list of Channels related to given Config.
	UpdateConnections(string, string, []string) error

	// List returns subset of Configs with given search params that belong to the
	// user identified by the given key.
	List(string, Filter, uint64, uint64) (ConfigsPage, error)

	// Remove removes Config with specified key that belongs to the user identified by the given key.
	Remove(string, string) error

	// Bootstrap returns Config to the Thing with provided external ID using external key.
	Bootstrap(string, string) (Config, error)

	// ChangeState changes state of the Thing with given ID and owner.
	ChangeState(string, string, State) error

	// Methods RemoveConfig, UpdateChannel, and RemoveChannel are used as
	// handlers for events. That's why these methods surpass ownership check.

	// RemoveConfigHandler removes Configuration with id received from an event.
	RemoveConfigHandler(string) error

	// UpdateChannelHandler updates Channel with data received from an event.
	UpdateChannelHandler(Channel) error

	// RemoveChannelHandler removes Channel with id received from an event.
	RemoveChannelHandler(string) error

	// DisconnectHandler changes state of the Config when connect/disconnect event occurs.
	DisconnectThingHandler(string, string) error
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
type ConfigReader interface {
	ReadConfig(Config) (mainflux.Response, error)
}

type bootstrapService struct {
	users   mainflux.UsersServiceClient
	configs ConfigRepository
	sdk     mfsdk.SDK
}

// New returns new Bootstrap service.
func New(users mainflux.UsersServiceClient, configs ConfigRepository, sdk mfsdk.SDK) Service {
	return &bootstrapService{
		configs: configs,
		sdk:     sdk,
		users:   users,
	}
}

func (bs bootstrapService) Add(key string, cfg Config) (Config, error) {
	owner, err := bs.identify(key)
	if err != nil {
		return Config{}, err
	}

	toConnect := bs.toIDList(cfg.MFChannels)

	// Check if channels exist. This is the way to prevent invalid configuration to be saved.
	existing, err := bs.configs.ListExisting(owner, toConnect)
	if err != nil {
		return Config{}, err
	}

	cfg.MFChannels, err = bs.connectionChannels(toConnect, bs.toIDList(existing), key)

	if err != nil {
		return Config{}, err
	}

	id := cfg.MFThing
	mfThing, err := bs.thing(key, id)
	if err != nil {
		return Config{}, err
	}

	cfg.MFThing = mfThing.ID
	cfg.Owner = owner
	cfg.State = Inactive
	cfg.MFKey = mfThing.Key
	saved, err := bs.configs.Save(cfg, toConnect)

	if err != nil {
		if id == "" {
			// Fail silently.
			bs.sdk.DeleteThing(cfg.MFThing, key)
		}
		return Config{}, err
	}

	cfg.MFThing = saved
	cfg.MFChannels = append(cfg.MFChannels, existing...)

	return cfg, nil
}

func (bs bootstrapService) View(key, id string) (Config, error) {
	owner, err := bs.identify(key)
	if err != nil {
		return Config{}, err
	}

	cfg, err := bs.configs.RetrieveByID(owner, id)
	if err != nil {
		return Config{}, err
	}

	for i, ch := range cfg.MFChannels {
		if meta, ok := ch.Metadata.([]byte); ok {
			if err := json.Unmarshal(meta, &cfg.MFChannels[i].Metadata); err != nil {
				return Config{}, err
			}
		}
	}

	return cfg, nil
}

func (bs bootstrapService) Update(key string, cfg Config) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}

	cfg.Owner = owner

	return bs.configs.Update(cfg)
}

func (bs bootstrapService) UpdateConnections(key, id string, connections []string) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}

	cfg, err := bs.configs.RetrieveByID(owner, id)
	if err != nil {
		return err
	}

	add, remove := bs.updateList(cfg, connections)
	channels, err := bs.updateChannels(add, key)
	if err != nil {
		return err
	}

	cfg.MFChannels = channels
	var connect, disconnect []string

	if cfg.State == Active {
		connect = add
		disconnect = remove
	}

	for _, c := range disconnect {
		if err := bs.sdk.DisconnectThing(id, c, key); err != nil {
			if err == mfsdk.ErrNotFound {
				continue
			}
			return ErrThings
		}
	}

	for _, c := range connect {
		if err := bs.sdk.ConnectThing(id, c, key); err != nil {
			if err == mfsdk.ErrNotFound {
				return ErrMalformedEntity
			}
			return ErrThings
		}
	}

	return bs.configs.UpdateConnections(owner, id, channels, connections)
}

func (bs bootstrapService) List(key string, filter Filter, offset, limit uint64) (ConfigsPage, error) {
	owner, err := bs.identify(key)
	if err != nil {
		return ConfigsPage{}, err
	}

	if filter.Unknown {
		return bs.configs.RetrieveUnknown(offset, limit), nil
	}

	return bs.configs.RetrieveAll(owner, filter, offset, limit), nil
}

func (bs bootstrapService) Remove(key, id string) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}

	return bs.configs.Remove(owner, id)
}

func (bs bootstrapService) Bootstrap(externalKey, externalID string) (Config, error) {
	cfg, err := bs.configs.RetrieveByExternalID(externalKey, externalID)
	if err != nil {
		if err == ErrNotFound {
			bs.configs.SaveUnknown(externalKey, externalID)
		}
		return Config{}, ErrNotFound
	}

	for i, ch := range cfg.MFChannels {
		if meta, ok := ch.Metadata.([]byte); ok {
			if err := json.Unmarshal(meta, &cfg.MFChannels[i].Metadata); err != nil {
				return Config{}, err
			}
		}
	}

	return cfg, nil
}

func (bs bootstrapService) ChangeState(key, id string, state State) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}

	thing, err := bs.configs.RetrieveByID(owner, id)
	if err != nil {
		return err
	}

	if thing.State == state {
		return nil
	}

	switch state {
	case Active:
		for _, c := range thing.MFChannels {
			if err := bs.sdk.ConnectThing(thing.MFThing, c.ID, key); err != nil {
				return ErrThings
			}
		}
	case Inactive:
		for _, c := range thing.MFChannels {
			if err := bs.sdk.DisconnectThing(thing.MFThing, c.ID, key); err != nil {
				if err == mfsdk.ErrNotFound {
					continue
				}
				return ErrThings
			}
		}
	}

	return bs.configs.ChangeState(owner, id, state)
}

func (bs bootstrapService) UpdateChannelHandler(channel Channel) error {
	return bs.configs.UpdateChannel(channel)
}

func (bs bootstrapService) RemoveConfigHandler(id string) error {
	return bs.configs.RemoveThing(id)
}

func (bs bootstrapService) RemoveChannelHandler(id string) error {
	return bs.configs.RemoveChannel(id)
}

func (bs bootstrapService) DisconnectThingHandler(channelID, thingID string) error {
	return bs.configs.DisconnectThing(channelID, thingID)
}

func (bs bootstrapService) identify(token string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := bs.users.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", ErrUnauthorizedAccess
	}

	return res.GetValue(), nil
}

// Method thing retrieves Mainflux Thing creating one if an empty ID is passed.
func (bs bootstrapService) thing(key, id string) (mfsdk.Thing, error) {
	thingID := id
	var err error

	if id == "" {
		thingID, err = bs.sdk.CreateThing(mfsdk.Thing{Type: thingType}, key)
		if err != nil {
			return mfsdk.Thing{}, err
		}
	}

	thing, err := bs.sdk.Thing(thingID, key)
	if err != nil {
		if err == mfsdk.ErrNotFound {
			return mfsdk.Thing{}, ErrNotFound
		}

		if id != "" {
			bs.sdk.DeleteThing(thingID, key)
		}

		return mfsdk.Thing{}, ErrThings
	}

	return thing, nil
}

// Method updateList accepts config and channel IDs and returns three lists:
// 1) IDs of Channels to be added
// 2) IDs of Channels to be removed
// 3) IDs of common Channels for these two configs
func (bs bootstrapService) updateList(cfg Config, connections []string) (add, remove []string) {
	var disconnect map[string]bool
	disconnect = make(map[string]bool, len(cfg.MFChannels))
	for _, c := range cfg.MFChannels {
		disconnect[c.ID] = true
	}

	for _, c := range connections {
		if disconnect[c] {
			// Don't disconnect common elements.
			delete(disconnect, c)
			continue
		}
		// Connect new elements.
		add = append(add, c)
	}

	for v := range disconnect {
		remove = append(remove, v)
	}

	return
}

func (bs bootstrapService) toIDList(channels []Channel) []string {
	var ret []string
	for _, ch := range channels {
		ret = append(ret, ch.ID)
	}

	return ret
}

func (bs bootstrapService) updateChannels(add []string, key string) ([]Channel, error) {
	var ret []Channel
	for _, id := range add {
		ch, err := bs.sdk.Channel(id, key)
		if err != nil {
			return []Channel{}, ErrMalformedEntity
		}

		ret = append(ret, Channel{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}

	return ret, nil
}

func (bs bootstrapService) connectionChannels(channels, existing []string, key string) ([]Channel, error) {
	add := make(map[string]bool, len(channels))
	for _, ch := range channels {
		add[ch] = true
	}

	for _, ch := range existing {
		if add[ch] == true {
			delete(add, ch)
		}
	}

	var ret []Channel
	for id := range add {
		ch, err := bs.sdk.Channel(id, key)
		if err != nil {
			return nil, ErrMalformedEntity
		}

		ret = append(ret, Channel{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}

	return ret, nil
}
