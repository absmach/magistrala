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
	"github.com/absmach/magistrala/auth"
	grpcclient "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policy"
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

	// ErrAddBootstrap indicates error in adding bootstrap configuration.
	ErrAddBootstrap = errors.New("failed to add bootstrap configuration")

	// ErrNotInSameDomain indicates entities are not in the same domain.
	errNotInSameDomain = errors.New("entities are not in the same domain")

	errUpdateConnections  = errors.New("failed to update connections")
	errRemoveBootstrap    = errors.New("failed to remove bootstrap configuration")
	errChangeState        = errors.New("failed to change state of bootstrap configuration")
	errUpdateChannel      = errors.New("failed to update channel")
	errRemoveConfig       = errors.New("failed to remove bootstrap configuration")
	errRemoveChannel      = errors.New("failed to remove channel")
	errCreateThing        = errors.New("failed to create thing")
	errConnectThing       = errors.New("failed to connect thing")
	errDisconnectThing    = errors.New("failed to disconnect thing")
	errCheckChannels      = errors.New("failed to check if channels exists")
	errConnectionChannels = errors.New("failed to check channels connections")
	errThingNotFound      = errors.New("failed to find thing")
	errUpdateCert         = errors.New("failed to update cert")
)

var _ Service = (*bootstrapService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
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

	// ChangeState changes state of the Thing with given thing ID and domain ID.
	ChangeState(ctx context.Context, token, id string, state State) error

	// Methods RemoveConfig, UpdateChannel, and RemoveChannel are used as
	// handlers for events. That's why these methods surpass ownership check.

	// UpdateChannelHandler updates Channel with data received from an event.
	UpdateChannelHandler(ctx context.Context, channel Channel) error

	// RemoveConfigHandler removes Configuration with id received from an event.
	RemoveConfigHandler(ctx context.Context, id string) error

	// RemoveChannelHandler removes Channel with id received from an event.
	RemoveChannelHandler(ctx context.Context, id string) error

	// ConnectThingHandler changes state of the Config to active when connect event occurs.
	ConnectThingHandler(ctx context.Context, channelID, ThingID string) error

	// DisconnectThingHandler changes state of the Config to inactive when disconnect event occurs.
	DisconnectThingHandler(ctx context.Context, channelID, ThingID string) error
}

// ConfigReader is used to parse Config into format which will be encoded
// as a JSON and consumed from the client side. The purpose of this interface
// is to provide convenient way to generate custom configuration response
// based on the specific Config which will be consumed by the client.
//
//go:generate mockery --name ConfigReader --output=./mocks --filename config_reader.go --quiet --note "Copyright (c) Abstract Machines"
type ConfigReader interface {
	ReadConfig(Config, bool) (interface{}, error)
}

type bootstrapService struct {
	auth       grpcclient.AuthServiceClient
	policy     policy.PolicyClient
	configs    ConfigRepository
	sdk        mgsdk.SDK
	encKey     []byte
	idProvider magistrala.IDProvider
}

// New returns new Bootstrap service.
func New(authClient grpcclient.AuthServiceClient, policyClient policy.PolicyClient, configs ConfigRepository, sdk mgsdk.SDK, encKey []byte, idp magistrala.IDProvider) Service {
	return &bootstrapService{
		configs:    configs,
		sdk:        sdk,
		auth:       authClient,
		policy:     policyClient,
		encKey:     encKey,
		idProvider: idp,
	}
}

func (bs bootstrapService) Add(ctx context.Context, token string, cfg Config) (Config, error) {
	user, err := bs.identify(ctx, token)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if _, err := bs.authorize(ctx, "", auth.UsersKind, user.GetId(), auth.MembershipPermission, auth.DomainType, user.GetDomainId()); err != nil {
		return Config{}, err
	}

	toConnect := bs.toIDList(cfg.Channels)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(ctx, user.GetDomainId(), toConnect)
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

	for _, channel := range cfg.Channels {
		if channel.DomainID != mgThing.DomainID {
			return Config{}, errors.Wrap(svcerr.ErrMalformedEntity, errNotInSameDomain)
		}
	}

	cfg.ThingID = mgThing.ID
	cfg.DomainID = user.GetDomainId()
	cfg.State = Inactive
	cfg.ThingKey = mgThing.Credentials.Secret

	saved, err := bs.configs.Save(ctx, cfg, toConnect)
	if err != nil {
		// If id is empty, then a new thing has been created function - bs.thing(id, token)
		// So, on bootstrap config save error , delete the newly created thing.
		if id == "" {
			if errT := bs.sdk.DeleteThing(cfg.ThingID, token); errT != nil {
				err = errors.Wrap(err, errT)
			}
		}
		return Config{}, errors.Wrap(ErrAddBootstrap, err)
	}

	cfg.ThingID = saved
	cfg.Channels = append(cfg.Channels, existing...)

	return cfg, nil
}

func (bs bootstrapService) View(ctx context.Context, token, id string) (Config, error) {
	user, err := bs.identify(ctx, token)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if _, err := bs.authorize(ctx, user.GetDomainId(), auth.UsersKind, user.GetId(), auth.ViewPermission, auth.ThingType, id); err != nil {
		return Config{}, err
	}
	cfg, err := bs.configs.RetrieveByID(ctx, user.GetDomainId(), id)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return cfg, nil
}

func (bs bootstrapService) Update(ctx context.Context, token string, cfg Config) error {
	user, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if _, err := bs.authorize(ctx, user.GetDomainId(), auth.UsersKind, user.GetId(), auth.EditPermission, auth.ThingType, cfg.ThingID); err != nil {
		return err
	}

	cfg.DomainID = user.GetDomainId()
	if err = bs.configs.Update(ctx, cfg); err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}
	return nil
}

func (bs bootstrapService) UpdateCert(ctx context.Context, token, thingID, clientCert, clientKey, caCert string) (Config, error) {
	user, err := bs.identify(ctx, token)
	if err != nil {
		return Config{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if _, err := bs.authorize(ctx, user.GetDomainId(), auth.UsersKind, user.GetId(), auth.EditPermission, auth.ThingType, thingID); err != nil {
		return Config{}, err
	}

	cfg, err := bs.configs.UpdateCert(ctx, user.GetDomainId(), thingID, clientCert, clientKey, caCert)
	if err != nil {
		return Config{}, errors.Wrap(errUpdateCert, err)
	}
	return cfg, nil
}

func (bs bootstrapService) UpdateConnections(ctx context.Context, token, id string, connections []string) error {
	user, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}

	if _, err := bs.authorize(ctx, user.GetDomainId(), auth.UsersKind, user.GetId(), auth.EditPermission, auth.ThingType, id); err != nil {
		return err
	}

	cfg, err := bs.configs.RetrieveByID(ctx, user.GetDomainId(), id)
	if err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}

	add, remove := bs.updateList(cfg, connections)

	// Check if channels exist. This is the way to prevent fetching channels that already exist.
	existing, err := bs.configs.ListExisting(ctx, user.GetDomainId(), connections)
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
			if errors.Contains(err, repoerr.ErrNotFound) {
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
	if err := bs.configs.UpdateConnections(ctx, user.GetDomainId(), id, channels, connections); err != nil {
		return errors.Wrap(errUpdateConnections, err)
	}
	return nil
}

func (bs bootstrapService) listClientIDs(ctx context.Context, userID string) ([]string, error) {
	tids, err := bs.policy.ListAllObjects(ctx, policy.PolicyReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  auth.ViewPermission,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (bs bootstrapService) checkSuperAdmin(ctx context.Context, userID string) error {
	res, err := bs.auth.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  auth.AdminPermission,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})
	if err != nil {
		return err
	}
	if !res.Authorized {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}

func (bs bootstrapService) List(ctx context.Context, token string, filter Filter, offset, limit uint64) (ConfigsPage, error) {
	user, err := bs.identify(ctx, token)
	if err != nil {
		return ConfigsPage{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	if err := bs.checkSuperAdmin(ctx, user.GetId()); err == nil {
		return bs.configs.RetrieveAll(ctx, user.GetDomainId(), []string{}, filter, offset, limit), nil
	}

	if _, err := bs.authorize(ctx, "", auth.UsersKind, user.GetId(), auth.AdminPermission, auth.DomainType, user.GetDomainId()); err == nil {
		return bs.configs.RetrieveAll(ctx, user.GetDomainId(), []string{}, filter, offset, limit), nil
	}

	// Handle non-admin users
	thingIDs, err := bs.listClientIDs(ctx, user.GetId())
	if err != nil {
		return ConfigsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}

	if len(thingIDs) == 0 {
		return ConfigsPage{
			Total:   0,
			Offset:  offset,
			Limit:   limit,
			Configs: []Config{},
		}, nil
	}

	return bs.configs.RetrieveAll(ctx, user.GetDomainId(), thingIDs, filter, offset, limit), nil
}

func (bs bootstrapService) Remove(ctx context.Context, token, id string) error {
	user, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if _, err := bs.authorize(ctx, user.GetDomainId(), auth.UsersKind, user.GetId(), auth.DeletePermission, auth.ThingType, id); err != nil {
		return err
	}
	if err := bs.configs.Remove(ctx, user.GetDomainId(), id); err != nil {
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
	user, err := bs.identify(ctx, token)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}

	cfg, err := bs.configs.RetrieveByID(ctx, user.GetDomainId(), id)
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
				// Ignore conflict errors as they indicate the connection already exists.
				if errors.Contains(err, svcerr.ErrConflict) {
					continue
				}
				return ErrThings
			}
		}
	case Inactive:
		for _, c := range cfg.Channels {
			if err := bs.sdk.DisconnectThing(cfg.ThingID, c.ID, token); err != nil {
				if errors.Contains(err, repoerr.ErrNotFound) {
					continue
				}
				return ErrThings
			}
		}
	}
	if err := bs.configs.ChangeState(ctx, user.GetDomainId(), id, state); err != nil {
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

func (bs bootstrapService) ConnectThingHandler(ctx context.Context, channelID, thingID string) error {
	if err := bs.configs.ConnectThing(ctx, channelID, thingID); err != nil {
		return errors.Wrap(errConnectThing, err)
	}
	return nil
}

func (bs bootstrapService) DisconnectThingHandler(ctx context.Context, channelID, thingID string) error {
	if err := bs.configs.DisconnectThing(ctx, channelID, thingID); err != nil {
		return errors.Wrap(errDisconnectThing, err)
	}
	return nil
}

func (bs bootstrapService) identify(ctx context.Context, token string) (*magistrala.IdentityRes, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	res, err := bs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if res.GetId() == "" || res.GetDomainId() == "" {
		return nil, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return res, nil
}

func (bs bootstrapService) authorize(ctx context.Context, domainID, subjKind, subj, perm, objType, obj string) (string, error) {
	req := &magistrala.AuthorizeReq{
		Domain:      domainID,
		SubjectType: auth.UserType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	res, err := bs.auth.Authorize(ctx, req)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return res.GetId(), nil
}

// Method thing retrieves Magistrala Thing creating one if an empty ID is passed.
func (bs bootstrapService) thing(id, token string) (mgsdk.Thing, error) {
	// If Thing ID is not provided, then create new thing.
	if id == "" {
		id, err := bs.idProvider.ID()
		if err != nil {
			return mgsdk.Thing{}, errors.Wrap(errCreateThing, err)
		}
		thing, sdkErr := bs.sdk.CreateThing(mgsdk.Thing{ID: id, Name: "Bootstrapped Thing " + id}, token)
		if sdkErr != nil {
			return mgsdk.Thing{}, errors.Wrap(errCreateThing, sdkErr)
		}
		return thing, nil
	}

	// If Thing ID is provided, then retrieve thing
	thing, sdkErr := bs.sdk.Thing(id, token)
	if sdkErr != nil {
		return mgsdk.Thing{}, errors.Wrap(ErrThings, sdkErr)
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
			DomainID: ch.DomainID,
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
