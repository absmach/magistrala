// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/errors"
	mfsdk "github.com/mainflux/mainflux/sdk/go"
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
	ErrThings = errors.New("failed to receive response from Things service")

	// ErrExternalKeyNotFound indicates a non-existent bootstrap configuration for given external key
	ErrExternalKeyNotFound = errors.New("failed to get bootstrap configuration for given external key")

	// ErrSecureBootstrap indicates erron in getting bootstrap configuration for given encrypted external key
	ErrSecureBootstrap = errors.New("failed to get bootstrap configuration for given encrypted external key")

	errAddBootstrap       = errors.New("failed to add bootstrap configuration")
	errUpdateConnections  = errors.New("failed to update connections")
	errRemoveBootstrap    = errors.New("failed to remove bootstrap configuration")
	errBootstrap          = errors.New("failed to read bootstrap configuration")
	errChangeState        = errors.New("failed to change state of bootstrap configuration")
	errUpdateChannel      = errors.New("failed to update channel")
	errRemoveConfig       = errors.New("failed to remove bootstrap configuration")
	errRemoveChannel      = errors.New("failed to remove channel")
	errCreateThing        = errors.New("failed to create thing")
	errDisconnectThing    = errors.New("failed to disconnect thing")
	errThingNotFound      = errors.New("thing not found")
	errCheckChannels      = errors.New("failed to check if channels exists")
	errConnectionChannels = errors.New("failed to check channels connections")
	errUpdateCert         = errors.New("failed to update cert")
)

var _ Service = (*bootstrapService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Add adds new Thing Config to the user identified by the provided key.
	Add(key string, cfg Config) (Config, error)

	// View returns Thing Config with given ID belonging to the user identified by the given key.
	View(key, id string) (Config, error)

	// Update updates editable fields of the provided Config.
	Update(key string, cfg Config) error

	// UpdateCert updates an existing Config certificate and key.
	// A non-nil error is returned to indicate operation failure.
	UpdateCert(key, thingID, clientCert, clientKey, caCert string) error

	// UpdateConnections updates list of Channels related to given Config.
	UpdateConnections(key, id string, connections []string) error

	// List returns subset of Configs with given search params that belong to the
	// user identified by the given key.
	List(key string, filter Filter, offset, limit uint64) (ConfigsPage, error)

	// Remove removes Config with specified key that belongs to the user identified by the given key.
	Remove(key, id string) error

	// Bootstrap returns Config to the Thing with provided external ID using external key.
	Bootstrap(externalKey, externalID string, secure bool) (Config, error)

	// ChangeState changes state of the Thing with given ID and owner.
	ChangeState(key, id string, state State) error

	// Methods RemoveConfig, UpdateChannel, and RemoveChannel are used as
	// handlers for events. That's why these methods surpass ownership check.

	// UpdateChannelHandler updates Channel with data received from an event.
	UpdateChannelHandler(channel Channel) error

	// RemoveConfigHandler removes Configuration with id received from an event.
	RemoveConfigHandler(id string) error

	// RemoveChannelHandler removes Channel with id received from an event.
	RemoveChannelHandler(id string) error

	// DisconnectHandler changes state of the Config when connect/disconnect event occurs.
	DisconnectThingHandler(channelID, thingID string) error
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
type ConfigReader interface {
	ReadConfig(Config, bool) (interface{}, error)
}

type bootstrapService struct {
	auth    mainflux.AuthNServiceClient
	configs ConfigRepository
	sdk     mfsdk.SDK
	encKey  []byte
	reader  ConfigReader
}

// New returns new Bootstrap service.
func New(auth mainflux.AuthNServiceClient, configs ConfigRepository, sdk mfsdk.SDK, encKey []byte) Service {
	return &bootstrapService{
		configs: configs,
		sdk:     sdk,
		auth:    auth,
		encKey:  encKey,
	}
}

func (bs bootstrapService) Add(key string, cfg Config) (Config, error) {
	owner, err := bs.identify(key)
	if err != nil {
		return Config{}, err
	}

	toConnect := bs.toIDList(cfg.MFChannels)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(owner, toConnect)
	if err != nil {
		return Config{}, errors.Wrap(errCheckChannels, err)
	}

	cfg.MFChannels, err = bs.connectionChannels(toConnect, bs.toIDList(existing), key)

	if err != nil {
		return Config{}, errors.Wrap(errConnectionChannels, err)
	}

	id := cfg.MFThing
	mfThing, err := bs.thing(key, id)
	if err != nil {
		return Config{}, errors.Wrap(errAddBootstrap, err)
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
		return Config{}, errors.Wrap(errAddBootstrap, err)
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

	return bs.configs.RetrieveByID(owner, id)
}

func (bs bootstrapService) Update(key string, cfg Config) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}

	cfg.Owner = owner

	return bs.configs.Update(cfg)
}

func (bs bootstrapService) UpdateCert(key, thingID, clientCert, clientKey, caCert string) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}
	if err := bs.configs.UpdateCert(owner, thingID, clientCert, clientKey, caCert); err != nil {
		return errors.Wrap(errUpdateCert, err)
	}
	return nil
}

func (bs bootstrapService) UpdateConnections(key, id string, connections []string) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}

	cfg, err := bs.configs.RetrieveByID(owner, id)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	add, remove := bs.updateList(cfg, connections)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(owner, connections)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	channels, err := bs.connectionChannels(connections, bs.toIDList(existing), key)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
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
		conIDs := mfsdk.ConnectionIDs{
			ChannelIDs: []string{c},
			ThingIDs:   []string{id},
		}
		if err := bs.sdk.Connect(conIDs, key); err != nil {
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
	if err := bs.configs.Remove(owner, id); err != nil {
		return errors.Wrap(errRemoveBootstrap, err)
	}
	return nil
}

func (bs bootstrapService) Bootstrap(externalKey, externalID string, secure bool) (Config, error) {
	cfg, err := bs.configs.RetrieveByExternalID(externalID)
	if err != nil {
		if errors.Contains(err, ErrNotFound) {
			bs.configs.SaveUnknown(externalKey, externalID)
			return Config{}, ErrNotFound
		}
		return cfg, errors.Wrap(errBootstrap, err)
	}

	if secure {
		dec, err := bs.dec(externalKey)
		if err != nil {
			return Config{}, errors.Wrap(ErrSecureBootstrap, err)
		}
		externalKey = dec
	}

	if cfg.ExternalKey != externalKey {
		return Config{}, errors.Wrap(ErrExternalKeyNotFound, ErrNotFound)
	}

	return cfg, nil
}

func (bs bootstrapService) ChangeState(key, id string, state State) error {
	owner, err := bs.identify(key)
	if err != nil {
		return err
	}

	cfg, err := bs.configs.RetrieveByID(owner, id)
	if err != nil {
		return errors.Wrap(errChangeState, err)
	}

	if cfg.State == state {
		return nil
	}

	switch state {
	case Active:
		for _, c := range cfg.MFChannels {
			conIDs := mfsdk.ConnectionIDs{
				ChannelIDs: []string{c.ID},
				ThingIDs:   []string{cfg.MFThing},
			}
			if err := bs.sdk.Connect(conIDs, key); err != nil {
				return ErrThings
			}
		}
	case Inactive:
		for _, c := range cfg.MFChannels {
			if err := bs.sdk.DisconnectThing(cfg.MFThing, c.ID, key); err != nil {
				if err == mfsdk.ErrNotFound {
					continue
				}
				return ErrThings
			}
		}
	}
	if err := bs.configs.ChangeState(owner, id, state); err != nil {
		return errors.Wrap(errChangeState, err)
	}
	return nil
}

func (bs bootstrapService) UpdateChannelHandler(channel Channel) error {
	if err := bs.configs.UpdateChannel(channel); err != nil {
		return errors.Wrap(errUpdateChannel, err)
	}
	return nil
}

func (bs bootstrapService) RemoveConfigHandler(id string) error {
	if err := bs.configs.RemoveThing(id); err != nil {
		return errors.Wrap(errRemoveConfig, err)
	}
	return nil
}

func (bs bootstrapService) RemoveChannelHandler(id string) error {
	if err := bs.configs.RemoveChannel(id); err != nil {
		return errors.Wrap(errRemoveChannel, err)
	}
	return nil
}

func (bs bootstrapService) DisconnectThingHandler(channelID, thingID string) error {
	if err := bs.configs.DisconnectThing(channelID, thingID); err != nil {
		return errors.Wrap(errDisconnectThing, err)
	}
	return nil
}

func (bs bootstrapService) identify(token string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := bs.auth.Identify(ctx, &mainflux.Token{Value: token})
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
		thingID, err = bs.sdk.CreateThing(mfsdk.Thing{}, key)
		if err != nil {
			return mfsdk.Thing{}, errors.Wrap(errCreateThing, err)
		}
	}

	thing, err := bs.sdk.Thing(thingID, key)
	if err != nil {
		if err == mfsdk.ErrNotFound {
			return mfsdk.Thing{}, errors.Wrap(errThingNotFound, ErrNotFound)
		}

		if id != "" {
			bs.sdk.DeleteThing(thingID, key)
		}

		return mfsdk.Thing{}, ErrThings
	}

	return thing, nil
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
			return nil, errors.Wrap(ErrMalformedEntity, err)
		}

		ret = append(ret, Channel{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}

	return ret, nil
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

func (bs bootstrapService) dec(in string) (string, error) {
	ciphertext, err := hex.DecodeString(in)
	if err != nil {
		return "", ErrNotFound
	}
	block, err := aes.NewCipher(bs.encKey)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aes.BlockSize {
		return "", ErrMalformedEntity
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext), nil
}
