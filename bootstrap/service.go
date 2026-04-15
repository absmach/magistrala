// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"

	"github.com/absmach/magistrala"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
)

var (
	connTypes = []string{"Publish", "Subscribe"}

	// ErrClients indicates failure to communicate with Magistrala Clients service.
	// It can be due to networking error or invalid/unauthenticated request.
	ErrClients = errors.New("failed to receive response from Clients service")

	// ErrExternalKey indicates a non-existent bootstrap configuration for given external key.
	ErrExternalKey = errors.NewAuthZError("failed to get bootstrap configuration for given external key")

	// ErrExternalKeySecure indicates error in getting bootstrap configuration for given encrypted external key.
	ErrExternalKeySecure = errors.NewAuthZError("failed to get bootstrap configuration for given encrypted external key")

	// ErrBootstrap indicates error in getting bootstrap configuration.
	ErrBootstrap = errors.New("failed to read bootstrap configuration")

	// ErrAddBootstrap indicates error in adding bootstrap configuration.
	ErrAddBootstrap = errors.NewServiceError("failed to add bootstrap configuration")

	// ErrBootstrapState indicates an invalid bootstrap state.
	ErrBootstrapState = errors.NewRequestError("invalid bootstrap state")

	// ErrNotInSameDomain indicates entities are not in the same domain.
	errNotInSameDomain = errors.New("entities are not in the same domain")

	errUpdateConnections  = errors.New("failed to update connections")
	errRemoveBootstrap    = errors.New("failed to remove bootstrap configuration")
	errChangeState        = errors.New("failed to change state of bootstrap configuration")
	errUpdateChannel      = errors.New("failed to update channel")
	errRemoveConfig       = errors.New("failed to remove bootstrap configuration")
	errRemoveChannel      = errors.New("failed to remove channel")
	errCreateClient       = errors.New("failed to create client")
	errConnectClient      = errors.New("failed to connect client")
	errDisconnectClient   = errors.New("failed to disconnect client")
	errCheckChannels      = errors.New("failed to check if channels exists")
	errConnectionChannels = errors.New("failed to check channels connections")
	errClientNotFound     = errors.New("failed to find client")
	errUpdateCert         = errors.New("failed to update cert")
)

var _ Service = (*bootstrapService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Add adds new Client Config to the user identified by the provided token.
	Add(ctx context.Context, session smqauthn.Session, token string, cfg Config) (Config, error)

	// View returns Client Config with given ID belonging to the user identified by the given token.
	View(ctx context.Context, session smqauthn.Session, id string) (Config, error)

	// Update updates editable fields of the provided Config.
	Update(ctx context.Context, session smqauthn.Session, cfg Config) error

	// UpdateCert updates an existing Config certificate and token.
	// A non-nil error is returned to indicate operation failure.
	UpdateCert(ctx context.Context, session smqauthn.Session, clientID, clientCert, clientKey, caCert string) (Config, error)

	// UpdateConnections updates list of Channels related to given Config.
	UpdateConnections(ctx context.Context, session smqauthn.Session, token, id string, connections []string) error

	// List returns subset of Configs with given search params that belong to the
	// user identified by the given token.
	List(ctx context.Context, session smqauthn.Session, filter Filter, offset, limit uint64) (ConfigsPage, error)

	// Remove removes Config with specified token that belongs to the user identified by the given token.
	Remove(ctx context.Context, session smqauthn.Session, id string) error

	// Bootstrap returns Config to the Client with provided external ID using external key.
	Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (Config, error)

	// ChangeState changes state of the Client with given client ID and domain ID.
	ChangeState(ctx context.Context, session smqauthn.Session, token, id string, state State) error

	// Methods RemoveConfig, UpdateChannel, and RemoveChannel are used as
	// handlers for events. That's why these methods surpass ownership check.

	// UpdateChannelHandler updates Channel with data received from an event.
	UpdateChannelHandler(ctx context.Context, channel Channel) error

	// RemoveConfigHandler removes Configuration with id received from an event.
	RemoveConfigHandler(ctx context.Context, id string) error

	// RemoveChannelHandler removes Channel with id received from an event.
	RemoveChannelHandler(ctx context.Context, id string) error

	// ConnectClientHandler changes state of the Config to active when connect event occurs.
	ConnectClientHandler(ctx context.Context, channelID, clientID string) error

	// DisconnectClientHandler changes state of the Config to inactive when disconnect event occurs.
	DisconnectClientHandler(ctx context.Context, channelID, clientID string) error
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
type ConfigReader interface {
	ReadConfig(Config, bool) (any, error)
}

type bootstrapService struct {
	policies   policies.Service
	configs    ConfigRepository
	sdk        mgsdk.SDK
	encKey     []byte
	idProvider magistrala.IDProvider
}

// New returns new Bootstrap service.
func New(policyService policies.Service, configs ConfigRepository, sdk mgsdk.SDK, encKey []byte, idp magistrala.IDProvider) Service {
	return &bootstrapService{
		configs:    configs,
		sdk:        sdk,
		policies:   policyService,
		encKey:     encKey,
		idProvider: idp,
	}
}

func (bs bootstrapService) Add(ctx context.Context, session smqauthn.Session, token string, cfg Config) (Config, error) {
	toConnect := bs.toIDList(cfg.Channels)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(ctx, session.DomainID, toConnect)
	if err != nil {
		return Config{}, errors.Wrap(errCheckChannels, err)
	}

	cfg.Channels, err = bs.connectionChannels(ctx, toConnect, bs.toIDList(existing), session.DomainID, token)
	if err != nil {
		return Config{}, errors.Wrap(errConnectionChannels, err)
	}

	id := cfg.ClientID
	mgClient, err := bs.client(ctx, session.DomainID, id, token)
	if err != nil {
		return Config{}, errors.Wrap(errClientNotFound, err)
	}

	for _, channel := range cfg.Channels {
		if channel.DomainID != mgClient.DomainID {
			return Config{}, errors.Wrap(svcerr.ErrMalformedEntity, errNotInSameDomain)
		}
	}

	cfg.ClientID = mgClient.ID
	cfg.DomainID = session.DomainID
	cfg.ClientSecret = mgClient.Credentials.Secret
	cfg.State = Inactive

	// Create actual connections via SDK if there are channels to connect
	if len(toConnect) > 0 {
		for _, channelID := range toConnect {
			if err := bs.sdk.ConnectClients(ctx, channelID, []string{cfg.ClientID}, connTypes, session.DomainID, token); err != nil {
				if errors.Contains(err, svcerr.ErrConflict) {
					continue
				}
				return Config{}, ErrClients
			}
		}
		cfg.State = Active
	}

	saved, err := bs.configs.Save(ctx, cfg, toConnect)
	if err != nil {
		// If id is empty, then a new client has been created function - bs.client(id, token)
		// So, on bootstrap config save error , delete the newly created client.
		if id == "" {
			if errT := bs.sdk.DeleteClient(ctx, cfg.ClientID, cfg.DomainID, token); errT != nil {
				err = errors.Wrap(err, errT)
			}
		}
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}

	cfg.ClientID = saved
	cfg.Channels = append(cfg.Channels, existing...)

	return cfg, nil
}

func (bs bootstrapService) View(ctx context.Context, session smqauthn.Session, id string) (Config, error) {
	cfg, err := bs.configs.RetrieveByID(ctx, session.DomainID, id)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return cfg, nil
}

func (bs bootstrapService) Update(ctx context.Context, session smqauthn.Session, cfg Config) error {
	cfg.DomainID = session.DomainID
	if err := bs.configs.Update(ctx, cfg); err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}
	return nil
}

func (bs bootstrapService) UpdateCert(ctx context.Context, session smqauthn.Session, clientID, clientCert, clientKey, caCert string) (Config, error) {
	cfg, err := bs.configs.UpdateCert(ctx, session.DomainID, clientID, clientCert, clientKey, caCert)
	if err != nil {
		return Config{}, errors.Wrap(errUpdateCert, err)
	}
	return cfg, nil
}

func (bs bootstrapService) UpdateConnections(ctx context.Context, session smqauthn.Session, token, id string, connections []string) error {
	cfg, err := bs.configs.RetrieveByID(ctx, session.DomainID, id)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}
	currentChannels := bs.toIDList(cfg.Channels)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(ctx, session.DomainID, connections)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	channels, err := bs.connectionChannels(ctx, connections, bs.toIDList(existing), session.DomainID, token)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	cfg.Channels = channels

	nextState := Active
	if cfg.State == Active {
		nextState = Inactive
	}

	switch nextState {
	case Active:
		if err := bs.connectChannels(ctx, session.DomainID, token, id, connections); err != nil {
			return ErrClients
		}
	case Inactive:
		disconnect := uniqueStrings(append(currentChannels, connections...))
		if err := bs.disconnectChannels(ctx, session.DomainID, token, id, disconnect); err != nil {
			return ErrClients
		}
	}

	if err := bs.configs.UpdateConnections(ctx, session.DomainID, id, channels, connections); err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	if err := bs.configs.ChangeState(ctx, session.DomainID, id, nextState); err != nil {
		return errors.Wrap(errChangeState, err)
	}
	return nil
}

func (bs bootstrapService) listClientIDs(ctx context.Context, userID string) ([]string, error) {
	tids, err := bs.policies.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  policies.ViewPermission,
		ObjectType:  policies.ClientType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (bs bootstrapService) List(ctx context.Context, session smqauthn.Session, filter Filter, offset, limit uint64) (ConfigsPage, error) {
	if session.SuperAdmin {
		return bs.configs.RetrieveAll(ctx, session.DomainID, []string{}, filter, offset, limit), nil
	}

	// Handle non-admin users
	clientIDs, err := bs.listClientIDs(ctx, session.DomainUserID)
	if err != nil {
		return ConfigsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}

	if len(clientIDs) == 0 {
		return ConfigsPage{
			Total:   0,
			Offset:  offset,
			Limit:   limit,
			Configs: []Config{},
		}, nil
	}

	return bs.configs.RetrieveAll(ctx, session.DomainID, clientIDs, filter, offset, limit), nil
}

func (bs bootstrapService) Remove(ctx context.Context, session smqauthn.Session, id string) error {
	if err := bs.configs.Remove(ctx, session.DomainID, id); err != nil {
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

func (bs bootstrapService) ChangeState(ctx context.Context, session smqauthn.Session, token, id string, state State) error {
	cfg, err := bs.configs.RetrieveByID(ctx, session.DomainID, id)
	if err != nil {
		return errors.Wrap(errChangeState, err)
	}

	if cfg.State == state {
		return nil
	}

	switch state {
	case Active:
		if err := bs.connectChannels(ctx, session.DomainID, token, cfg.ClientID, bs.toIDList(cfg.Channels)); err != nil {
			return ErrClients
		}
	case Inactive:
		if err := bs.disconnectChannels(ctx, session.DomainID, token, cfg.ClientID, bs.toIDList(cfg.Channels)); err != nil {
			return ErrClients
		}
	}
	if err := bs.configs.ChangeState(ctx, session.DomainID, id, state); err != nil {
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
	if err := bs.configs.RemoveClient(ctx, id); err != nil {
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

func (bs bootstrapService) ConnectClientHandler(ctx context.Context, channelID, clientID string) error {
	if err := bs.configs.ConnectClient(ctx, channelID, clientID); err != nil {
		return errors.Wrap(errConnectClient, err)
	}
	return nil
}

func (bs bootstrapService) DisconnectClientHandler(ctx context.Context, channelID, clientID string) error {
	if err := bs.configs.DisconnectClient(ctx, channelID, clientID); err != nil {
		return errors.Wrap(errDisconnectClient, err)
	}
	return nil
}

func (bs bootstrapService) connectChannels(ctx context.Context, domainID, token, clientID string, channelIDs []string) error {
	return bs.sdk.Connect(ctx, mgsdk.Connection{
		ChannelIDs: channelIDs,
		ClientIDs:  []string{clientID},
		Types:      connTypes,
	}, domainID, token)
}

func (bs bootstrapService) disconnectChannels(ctx context.Context, domainID, token, clientID string, channelIDs []string) error {
	return bs.sdk.Disconnect(ctx, mgsdk.Connection{
		ChannelIDs: channelIDs,
		ClientIDs:  []string{clientID},
		Types:      connTypes,
	}, domainID, token)
}

// Method client retrieves Magistrala Client creating one if an empty ID is passed.
func (bs bootstrapService) client(ctx context.Context, domainID, id, token string) (mgsdk.Client, error) {
	// If Client ID is not provided, then create new client.
	if id == "" {
		id, err := bs.idProvider.ID()
		if err != nil {
			return mgsdk.Client{}, errors.Wrap(errCreateClient, err)
		}
		client, sdkErr := bs.sdk.CreateClient(ctx, mgsdk.Client{ID: id, Name: "Bootstrapped Client " + id}, domainID, token)
		if sdkErr != nil {
			return mgsdk.Client{}, errors.Wrap(errCreateClient, sdkErr)
		}
		return client, nil
	}
	// If Client ID is provided, then retrieve client
	client, sdkErr := bs.sdk.Client(ctx, id, domainID, token)
	if sdkErr != nil {
		return mgsdk.Client{}, errors.Wrap(ErrClients, sdkErr)
	}
	return client, nil
}

func (bs bootstrapService) connectionChannels(ctx context.Context, channels, existing []string, domainID, token string) ([]Channel, error) {
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
		ch, err := bs.sdk.Channel(ctx, id, domainID, token)
		if err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}

		ret = append(ret, Channel{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
			DomainID: ch.DomainID,
		})
	}

	return ret, nil
}

func (bs bootstrapService) toIDList(channels []Channel) []string {
	var ret []string
	for _, ch := range channels {
		ret = append(ret, ch.ID)
	}

	return ret
}

func uniqueStrings(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	ret := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ret = append(ret, id)
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
