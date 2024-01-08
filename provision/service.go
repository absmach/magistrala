// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"encoding/json"
	"fmt"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
)

const (
	externalIDKey = "external_id"
	gateway       = "gateway"
	Active        = 1

	control = "control"
	data    = "data"
	export  = "export"
)

var (
	ErrUnauthorized             = errors.New("unauthorized access")
	ErrFailedToCreateToken      = errors.New("failed to create access token")
	ErrEmptyThingsList          = errors.New("things list in configuration empty")
	ErrThingUpdate              = errors.New("failed to update thing")
	ErrEmptyChannelsList        = errors.New("channels list in configuration is empty")
	ErrFailedChannelCreation    = errors.New("failed to create channel")
	ErrFailedChannelRetrieval   = errors.New("failed to retrieve channel")
	ErrFailedThingCreation      = errors.New("failed to create thing")
	ErrFailedThingRetrieval     = errors.New("failed to retrieve thing")
	ErrMissingCredentials       = errors.New("missing credentials")
	ErrFailedBootstrapRetrieval = errors.New("failed to retrieve bootstrap")
	ErrFailedCertCreation       = errors.New("failed to create certificates")
	ErrFailedBootstrap          = errors.New("failed to create bootstrap config")
	ErrFailedBootstrapValidate  = errors.New("failed to validate bootstrap config creation")
	ErrGatewayUpdate            = errors.New("failed to updated gateway metadata")

	limit  uint = 10
	offset uint = 0
)

var _ Service = (*provisionService)(nil)

// Service specifies Provision service API.
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// Provision is the only method this API specifies. Depending on the configuration,
	// the following actions will can be executed:
	// - create a Thing based on external_id (eg. MAC address)
	// - create multiple Channels
	// - create Bootstrap configuration
	// - whitelist Thing in Bootstrap configuration == connect Thing to Channels
	Provision(token, name, externalID, externalKey string) (Result, error)

	// Mapping returns current configuration used for provision
	// useful for using in ui to create configuration that matches
	// one created with Provision method.
	Mapping(token string) (map[string]interface{}, error)

	// Certs creates certificate for things that communicate over mTLS
	// A duration string is a possibly signed sequence of decimal numbers,
	// each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	Cert(token, thingID, duration string) (string, string, error)
}

type provisionService struct {
	logger mglog.Logger
	sdk    sdk.SDK
	conf   Config
}

// Result represent what is created with additional info.
type Result struct {
	Things      []sdk.Thing       `json:"things,omitempty"`
	Channels    []sdk.Channel     `json:"channels,omitempty"`
	ClientCert  map[string]string `json:"client_cert,omitempty"`
	ClientKey   map[string]string `json:"client_key,omitempty"`
	CACert      string            `json:"ca_cert,omitempty"`
	Whitelisted map[string]bool   `json:"whitelisted,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// New returns new provision service.
func New(cfg Config, mgsdk sdk.SDK, logger mglog.Logger) Service {
	return &provisionService{
		logger: logger,
		conf:   cfg,
		sdk:    mgsdk,
	}
}

// Mapping retrieves current configuration.
func (ps *provisionService) Mapping(token string) (map[string]interface{}, error) {
	pm := sdk.PageMetadata{
		Offset: uint64(offset),
		Limit:  uint64(limit),
	}

	if _, err := ps.sdk.Users(pm, token); err != nil {
		return map[string]interface{}{}, errors.Wrap(ErrUnauthorized, err)
	}

	return ps.conf.Bootstrap.Content, nil
}

// Provision is provision method for creating setup according to
// provision layout specified in config.toml.
func (ps *provisionService) Provision(token, name, externalID, externalKey string) (res Result, err error) {
	var channels []sdk.Channel
	var things []sdk.Thing
	defer ps.recover(&err, &things, &channels, &token)

	token, err = ps.createTokenIfEmpty(token)
	if err != nil {
		return res, errors.Wrap(ErrFailedToCreateToken, err)
	}

	if len(ps.conf.Things) == 0 {
		return res, ErrEmptyThingsList
	}
	if len(ps.conf.Channels) == 0 {
		return res, ErrEmptyChannelsList
	}
	for _, thing := range ps.conf.Things {
		// If thing in configs contains metadata with external_id
		// set value for it from the provision request
		if _, ok := thing.Metadata[externalIDKey]; ok {
			thing.Metadata[externalIDKey] = externalID
		}

		th := sdk.Thing{
			Metadata: thing.Metadata,
		}
		if name == "" {
			name = thing.Name
		}
		th.Name = name
		th, err := ps.sdk.CreateThing(th, token)
		if err != nil {
			res.Error = err.Error()
			return res, errors.Wrap(ErrFailedThingCreation, err)
		}

		// Get newly created thing (in order to get the key).
		th, err = ps.sdk.Thing(th.ID, token)
		if err != nil {
			e := errors.Wrap(err, fmt.Errorf("thing id: %s", th.ID))
			return res, errors.Wrap(ErrFailedThingRetrieval, e)
		}
		things = append(things, th)
	}

	for _, channel := range ps.conf.Channels {
		ch := sdk.Channel{
			Name:     channel.Name,
			Metadata: sdk.Metadata(channel.Metadata),
		}
		ch, err := ps.sdk.CreateChannel(ch, token)
		if err != nil {
			return res, errors.Wrap(ErrFailedChannelCreation, err)
		}
		ch, err = ps.sdk.Channel(ch.ID, token)
		if err != nil {
			e := errors.Wrap(err, fmt.Errorf("channel id: %s", ch.ID))
			return res, errors.Wrap(ErrFailedChannelRetrieval, e)
		}
		channels = append(channels, ch)
	}

	res = Result{
		Things:      things,
		Channels:    channels,
		Whitelisted: map[string]bool{},
		ClientCert:  map[string]string{},
		ClientKey:   map[string]string{},
	}

	var cert sdk.Cert
	var bsConfig sdk.BootstrapConfig
	for _, thing := range things {
		var chanIDs []string

		for _, ch := range channels {
			chanIDs = append(chanIDs, ch.ID)
		}
		content, err := json.Marshal(ps.conf.Bootstrap.Content)
		if err != nil {
			return Result{}, errors.Wrap(ErrFailedBootstrap, err)
		}

		if ps.conf.Bootstrap.Provision && needsBootstrap(thing) {
			bsReq := sdk.BootstrapConfig{
				ThingID:     thing.ID,
				ExternalID:  externalID,
				ExternalKey: externalKey,
				Channels:    chanIDs,
				CACert:      res.CACert,
				ClientCert:  cert.ClientCert,
				ClientKey:   cert.ClientKey,
				Content:     string(content),
			}
			bsid, err := ps.sdk.AddBootstrap(bsReq, token)
			if err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrap, err)
			}

			bsConfig, err = ps.sdk.ViewBootstrap(bsid, token)
			if err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrapValidate, err)
			}
		}

		if ps.conf.Bootstrap.X509Provision {
			var cert sdk.Cert

			cert, err = ps.sdk.IssueCert(thing.ID, ps.conf.Cert.TTL, token)
			if err != nil {
				e := errors.Wrap(err, fmt.Errorf("thing id: %s", thing.ID))
				return res, errors.Wrap(ErrFailedCertCreation, e)
			}

			res.ClientCert[thing.ID] = cert.ClientCert
			res.ClientKey[thing.ID] = cert.ClientKey
			res.CACert = ""

			if needsBootstrap(thing) {
				if _, err = ps.sdk.UpdateBootstrapCerts(bsConfig.ThingID, cert.ClientCert, cert.ClientKey, "", token); err != nil {
					return Result{}, errors.Wrap(ErrFailedCertCreation, err)
				}
			}
		}

		if ps.conf.Bootstrap.AutoWhiteList {
			wlReq := sdk.BootstrapConfig{
				ThingID: thing.ID,
				State:   Active,
			}
			if err := ps.sdk.Whitelist(wlReq, token); err != nil {
				res.Error = err.Error()
				return res, ErrThingUpdate
			}
			res.Whitelisted[thing.ID] = true
		}
	}

	if err = ps.updateGateway(token, bsConfig, channels); err != nil {
		return res, err
	}
	return res, nil
}

func (ps *provisionService) Cert(token, thingID, ttl string) (string, string, error) {
	token, err := ps.createTokenIfEmpty(token)
	if err != nil {
		return "", "", errors.Wrap(ErrFailedToCreateToken, err)
	}

	th, err := ps.sdk.Thing(thingID, token)
	if err != nil {
		return "", "", errors.Wrap(ErrUnauthorized, err)
	}
	cert, err := ps.sdk.IssueCert(th.ID, ps.conf.Cert.TTL, token)
	return cert.ClientCert, cert.ClientKey, err
}

func (ps *provisionService) createTokenIfEmpty(token string) (string, error) {
	if token != "" {
		return token, nil
	}

	// If no token in request is provided
	// use API key provided in config file or env
	if ps.conf.Server.MgAPIKey != "" {
		return ps.conf.Server.MgAPIKey, nil
	}

	// If no API key use username and password provided to create access token.
	if ps.conf.Server.MgUser == "" || ps.conf.Server.MgPass == "" {
		return token, ErrMissingCredentials
	}

	u := sdk.Login{
		Identity: ps.conf.Server.MgUser,
		Secret:   ps.conf.Server.MgPass,
		DomainID: ps.conf.Server.MgDomainID,
	}
	tkn, err := ps.sdk.CreateToken(u)
	if err != nil {
		return token, errors.Wrap(ErrFailedToCreateToken, err)
	}

	return tkn.AccessToken, nil
}

func (ps *provisionService) updateGateway(token string, bs sdk.BootstrapConfig, channels []sdk.Channel) error {
	var gw Gateway
	for _, ch := range channels {
		switch ch.Metadata["type"] {
		case control:
			gw.CtrlChannelID = ch.ID
		case data:
			gw.DataChannelID = ch.ID
		case export:
			gw.ExportChannelID = ch.ID
		}
	}
	gw.ExternalID = bs.ExternalID
	gw.ExternalKey = bs.ExternalKey
	gw.CfgID = bs.ThingID
	gw.Type = gateway

	th, sdkerr := ps.sdk.Thing(bs.ThingID, token)
	if sdkerr != nil {
		return errors.Wrap(ErrGatewayUpdate, sdkerr)
	}
	b, err := json.Marshal(gw)
	if err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	if err := json.Unmarshal(b, &th.Metadata); err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	if _, err := ps.sdk.UpdateThing(th, token); err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	return nil
}

func (ps *provisionService) errLog(err error) {
	if err != nil {
		ps.logger.Error(fmt.Sprintf("Error recovering: %s", err))
	}
}

func clean(ps *provisionService, things []sdk.Thing, channels []sdk.Channel, token string) {
	for _, t := range things {
		_, err := ps.sdk.DisableThing(t.ID, token)
		ps.errLog(err)
	}
	for _, c := range channels {
		_, err := ps.sdk.DisableChannel(c.ID, token)
		ps.errLog(err)
	}
}

func (ps *provisionService) recover(e *error, ths *[]sdk.Thing, chs *[]sdk.Channel, tkn *string) {
	if e == nil {
		return
	}
	things, channels, token, err := *ths, *chs, *tkn, *e

	if errors.Contains(err, ErrFailedThingRetrieval) || errors.Contains(err, ErrFailedChannelCreation) {
		for _, th := range things {
			_, err := ps.sdk.DisableThing(th.ID, token)
			ps.errLog(err)
		}
		return
	}

	if errors.Contains(err, ErrFailedBootstrap) || errors.Contains(err, ErrFailedChannelRetrieval) {
		clean(ps, things, channels, token)
		return
	}

	if errors.Contains(err, ErrFailedBootstrapValidate) || errors.Contains(err, ErrFailedCertCreation) {
		clean(ps, things, channels, token)
		for _, th := range things {
			if needsBootstrap(th) {
				ps.errLog(ps.sdk.RemoveBootstrap(th.ID, token))
			}
		}
		return
	}

	if errors.Contains(err, ErrFailedBootstrapValidate) || errors.Contains(err, ErrFailedCertCreation) {
		clean(ps, things, channels, token)
		for _, th := range things {
			if needsBootstrap(th) {
				bs, err := ps.sdk.ViewBootstrap(th.ID, token)
				ps.errLog(errors.Wrap(ErrFailedBootstrapRetrieval, err))
				ps.errLog(ps.sdk.RemoveBootstrap(bs.ThingID, token))
			}
		}
	}

	if errors.Contains(err, ErrThingUpdate) || errors.Contains(err, ErrGatewayUpdate) {
		clean(ps, things, channels, token)
		for _, th := range things {
			if ps.conf.Bootstrap.X509Provision && needsBootstrap(th) {
				_, err := ps.sdk.RevokeCert(th.ID, token)
				ps.errLog(err)
			}
			if needsBootstrap(th) {
				bs, err := ps.sdk.ViewBootstrap(th.ID, token)
				ps.errLog(errors.Wrap(ErrFailedBootstrapRetrieval, err))
				ps.errLog(ps.sdk.RemoveBootstrap(bs.ThingID, token))
			}
		}
		return
	}
}

func needsBootstrap(th sdk.Thing) bool {
	if th.Metadata == nil {
		return false
	}

	if _, ok := th.Metadata[externalIDKey]; ok {
		return true
	}
	return false
}
