// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
)

var (
	// ErrThings indicates failure to communicate with Magistrala Things service.
	// It can be due to networking error or invalid/unauthenticated request.
	ErrThings = errors.New("failed to receive response from Things service")

	// ErrExternalKey indicates a non-existent bootstrap configuration for given external key.
	ErrExternalKey = errors.New("failed to get bootstrap configuration for given external key")

	// ErrExternalKeySecure indicates error in getting bootstrap configuration for given encrypted external key.
	ErrExternalKeySecure = errors.New("failed to get bootstrap configuration for given encrypted external key")

	// ErrBootstrap indicates error in getting bootstrap configuration.
	ErrBootstrap = errors.New("failed to read bootstrap configuration")

	errAddBootstrap       = errors.New("failed to add bootstrap configuration")
	errUpdateConnections  = errors.New("failed to update connections")
	errRemoveBootstrap    = errors.New("failed to remove bootstrap configuration")
	errChangeState        = errors.New("failed to change state of bootstrap configuration")
	errUpdateChannel      = errors.New("failed to update channel")
	errRemoveConfig       = errors.New("failed to remove bootstrap configuration")
	errRemoveChannel      = errors.New("failed to remove channel")
	errCreateThing        = errors.New("failed to create thing")
	errDisconnectThing    = errors.New("failed to disconnect thing")
	errCheckChannels      = errors.New("failed to check if channels exists")
	errConnectionChannels = errors.New("failed to check channels connections")
	errThingNotFound      = errors.New("failed to find thing")
	errUpdateCert         = errors.New("failed to update cert")
)

var _ Service = (*bootstrapService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Add adds new Thing Config to the user identified by the provided token.
	Add(ctx context.Context, token string, cfg Config) (Config, error)

	// View returns Thing Config with given ID belonging to the user identified by the given token.
	View(ctx context.Context, token, id string) (Config, error)

	// Update updates editable fields of the provided Config.
	Update(ctx context.Context, token string, cfg Config) error

	// UpdateCert updates an existing Config certificate and token.
	// A non-nil error is returned to indicate operation failure.
	UpdateCert(ctx context.Context, token, thingID, clientCert, clientKey, caCert string) (Config, error)

	// UpdateConnections updates list of Channels related to given Config.
	UpdateConnections(ctx context.Context, token, id string, connections []string) error

	// List returns subset of Configs with given search params that belong to the
	// user identified by the given token.
	List(ctx context.Context, token string, filter Filter, offset, limit uint64) (ConfigsPage, error)

	// Remove removes Config with specified token that belongs to the user identified by the given token.
	Remove(ctx context.Context, token, id string) error

	// Bootstrap returns Config to the Thing with provided external ID using external key.
	Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (Config, error)

	// ChangeState changes state of the Thing with given ID and owner.
	ChangeState(ctx context.Context, token, id string, state State) error

	// Methods RemoveConfig, UpdateChannel, and RemoveChannel are used as
	// handlers for events. That's why these methods surpass ownership check.

	// UpdateChannelHandler updates Channel with data received from an event.
	UpdateChannelHandler(ctx context.Context, channel Channel) error

	// RemoveConfigHandler removes Configuration with id received from an event.
	RemoveConfigHandler(ctx context.Context, id string) error

	// RemoveChannelHandler removes Channel with id received from an event.
	RemoveChannelHandler(ctx context.Context, id string) error

	// DisconnectHandler changes state of the Config when connect/disconnect event occurs.
	DisconnectThingHandler(ctx context.Context, channelID, thingID string) error
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
type ConfigReader interface {
	ReadConfig(Config, bool) (interface{}, error)
}

type bootstrapService struct {
	auth    magistrala.AuthServiceClient
	configs ConfigRepository
	sdk     mgsdk.SDK
	encKey  []byte
}

// New returns new Bootstrap service.
func New(auth magistrala.AuthServiceClient, configs ConfigRepository, sdk mgsdk.SDK, encKey []byte) Service {
	return &bootstrapService{
		configs: configs,
		sdk:     sdk,
		auth:    auth,
		encKey:  encKey,
	}
}

func (bs bootstrapService) Add(ctx context.Context, token string, cfg Config) (Config, error) {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	toConnect := bs.toIDList(cfg.Channels)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(ctx, owner, toConnect)
	if err != nil {
		return Config{}, errors.Wrap(errCheckChannels, err)
	}

	cfg.Channels, err = bs.connectionChannels(toConnect, bs.toIDList(existing), token)

	if err != nil {
		return Config{}, errors.Wrap(errConnectionChannels, err)
	}

	id := cfg.ThingID
	mgThing, err := bs.thing(id, token)
	if err != nil {
		return Config{}, errors.Wrap(errThingNotFound, err)
	}

	cfg.ThingID = mgThing.ID
	cfg.Owner = owner
	cfg.State = Inactive
	cfg.ThingKey = mgThing.Credentials.Secret

	saved, err := bs.configs.Save(ctx, cfg, toConnect)
	if err != nil {
		if id == "" {
			if _, errT := bs.sdk.DisableThing(cfg.ThingID, token); errT != nil {
				err = errors.Wrap(err, errT)
			}
		}
		return Config{}, errors.Wrap(errAddBootstrap, err)
	}

	cfg.ThingID = saved
	cfg.Channels = append(cfg.Channels, existing...)

	return cfg, nil
}

func (bs bootstrapService) View(ctx context.Context, token, id string) (Config, error) {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	cfg, err := bs.configs.RetrieveByID(ctx, owner, id)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return cfg, nil
}

func (bs bootstrapService) Update(ctx context.Context, token string, cfg Config) error {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}

	cfg.Owner = owner
	if err = bs.configs.Update(ctx, cfg); err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}
	return nil
}

func (bs bootstrapService) UpdateCert(ctx context.Context, token, thingID, clientCert, clientKey, caCert string) (Config, error) {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	cfg, err := bs.configs.UpdateCert(ctx, owner, thingID, clientCert, clientKey, caCert)
	if err != nil {
		return Config{}, errors.Wrap(errUpdateCert, err)
	}
	return cfg, nil
}

func (bs bootstrapService) UpdateConnections(ctx context.Context, token, id string, connections []string) error {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}

	cfg, err := bs.configs.RetrieveByID(ctx, owner, id)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	add, remove := bs.updateList(cfg, connections)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(ctx, owner, connections)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	channels, err := bs.connectionChannels(connections, bs.toIDList(existing), token)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	cfg.Channels = channels
	var connect, disconnect []string

	if cfg.State == Active {
		connect = add
		disconnect = remove
	}

	for _, c := range disconnect {
		if err := bs.sdk.DisconnectThing(id, c, token); err != nil {
			if errors.Contains(err, errors.ErrNotFound) {
				continue
			}
			return ErrThings
		}
	}

	for _, c := range connect {
		conIDs := mgsdk.Connection{
			ChannelID: c,
			ThingID:   id,
		}
		if err := bs.sdk.Connect(conIDs, token); err != nil {
			return ErrThings
		}
	}
	if err := bs.configs.UpdateConnections(ctx, owner, id, channels, connections); err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}
	return nil
}

func (bs bootstrapService) List(ctx context.Context, token string, filter Filter, offset, limit uint64) (ConfigsPage, error) {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return ConfigsPage{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return bs.configs.RetrieveAll(ctx, owner, filter, offset, limit), nil
}

func (bs bootstrapService) Remove(ctx context.Context, token, id string) error {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if err := bs.configs.Remove(ctx, owner, id); err != nil {
		return errors.Wrap(errRemoveBootstrap, err)
	}
	return nil
}

func (bs bootstrapService) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (Config, error) {
	cfg, err := bs.configs.RetrieveByExternalID(ctx, externalID)
	if err != nil {
		return cfg, errors.Wrap(ErrBootstrap, err)
	}
	if secure {
		dec, err := bs.dec(externalKey)
		if err != nil {
			return Config{}, errors.Wrap(ErrExternalKeySecure, err)
		}
		externalKey = dec
	}
	if cfg.ExternalKey != externalKey {
		return Config{}, ErrExternalKey
	}

	return cfg, nil
}

func (bs bootstrapService) ChangeState(ctx context.Context, token, id string, state State) error {
	owner, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}

	cfg, err := bs.configs.RetrieveByID(ctx, owner, id)
	if err != nil {
		return errors.Wrap(errChangeState, err)
	}

	if cfg.State == state {
		return nil
	}

	switch state {
	case Active:
		for _, c := range cfg.Channels {
			conIDs := mgsdk.Connection{
				ChannelID: c.ID,
				ThingID:   cfg.ThingID,
			}
			if err := bs.sdk.Connect(conIDs, token); err != nil {
				return ErrThings
			}
		}
	case Inactive:
		for _, c := range cfg.Channels {
			if err := bs.sdk.DisconnectThing(cfg.ThingID, c.ID, token); err != nil {
				if errors.Contains(err, errors.ErrNotFound) {
					continue
				}
				return ErrThings
			}
		}
	}
	if err := bs.configs.ChangeState(ctx, owner, id, state); err != nil {
		return errors.Wrap(errChangeState, err)
	}
	return nil
}

func (bs bootstrapService) UpdateChannelHandler(ctx context.Context, channel Channel) error {
	if err := bs.configs.UpdateChannel(ctx, channel); err != nil {
		return errors.Wrap(errUpdateChannel, err)
	}
	return nil
}

func (bs bootstrapService) RemoveConfigHandler(ctx context.Context, id string) error {
	if err := bs.configs.RemoveThing(ctx, id); err != nil {
		return errors.Wrap(errRemoveConfig, err)
	}
	return nil
}

func (bs bootstrapService) RemoveChannelHandler(ctx context.Context, id string) error {
	if err := bs.configs.RemoveChannel(ctx, id); err != nil {
		return errors.Wrap(errRemoveChannel, err)
	}
	return nil
}

func (bs bootstrapService) DisconnectThingHandler(ctx context.Context, channelID, thingID string) error {
	if err := bs.configs.DisconnectThing(ctx, channelID, thingID); err != nil {
		return errors.Wrap(errDisconnectThing, err)
	}
	return nil
}

func (bs bootstrapService) identify(ctx context.Context, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	res, err := bs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthentication, err)
	}

	return res.GetId(), nil
}

// Method thing retrieves Magistrala Thing creating one if an empty ID is passed.
func (bs bootstrapService) thing(id, token string) (mgsdk.Thing, error) {
	var thing mgsdk.Thing
	var err error
	var sdkErr errors.SDKError

	thing.ID = id
	if id == "" {
		thing, sdkErr = bs.sdk.CreateThing(mgsdk.Thing{}, token)
		if err != nil {
			return mgsdk.Thing{}, errors.Wrap(errCreateThing, errors.New(sdkErr.Err().Msg()))
		}
	}

	thing, sdkErr = bs.sdk.Thing(thing.ID, token)
	if sdkErr != nil {
		err = errors.New(sdkErr.Error())
		if id != "" {
			if _, sdkErr2 := bs.sdk.DisableThing(thing.ID, token); sdkErr2 != nil {
				err = errors.Wrap(errors.New(sdkErr.Msg()), errors.New(sdkErr2.Msg()))
			}
		}
		return mgsdk.Thing{}, errors.Wrap(ErrThings, err)
	}

	return thing, nil
}

func (bs bootstrapService) connectionChannels(channels, existing []string, token string) ([]Channel, error) {
	add := make(map[string]bool, len(channels))
	for _, ch := range channels {
		add[ch] = true
	}

	for _, ch := range existing {
		if add[ch] {
			delete(add, ch)
		}
	}

	var ret []Channel
	for id := range add {
		ch, err := bs.sdk.Channel(id, token)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
// 3) IDs of common Channels for these two configs.
func (bs bootstrapService) updateList(cfg Config, connections []string) (add, remove []string) {
	disconnect := make(map[string]bool, len(cfg.Channels))
	for _, c := range cfg.Channels {
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
		return "", err
	}
	block, err := aes.NewCipher(bs.encKey)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aes.BlockSize {
		return "", err
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext), nil
}
