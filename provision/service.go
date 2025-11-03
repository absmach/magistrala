// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	certs "github.com/absmach/certs"
	csdk "github.com/absmach/certs/sdk"
	"github.com/absmach/magistrala/pkg/sdk"
	"github.com/absmach/supermq/pkg/errors"
	smqSDK "github.com/absmach/supermq/pkg/sdk"
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
	ErrEmptyClientsList         = errors.New("clients list in configuration empty")
	ErrClientUpdate             = errors.New("failed to update client")
	ErrEmptyChannelsList        = errors.New("channels list in configuration is empty")
	ErrFailedChannelCreation    = errors.New("failed to create channel")
	ErrFailedChannelRetrieval   = errors.New("failed to retrieve channel")
	ErrFailedClientCreation     = errors.New("failed to create client")
	ErrFailedClientRetrieval    = errors.New("failed to retrieve client")
	ErrMissingCredentials       = errors.New("missing credentials")
	ErrFailedBootstrapRetrieval = errors.New("failed to retrieve bootstrap")
	ErrFailedCertCreation       = errors.New("failed to create certificates")
	ErrFailedCertView           = errors.New("failed to view certificate")
	ErrFailedBootstrap          = errors.New("failed to create bootstrap config")
	ErrFailedBootstrapValidate  = errors.New("failed to validate bootstrap config creation")
	ErrGatewayUpdate            = errors.New("failed to updated gateway metadata")

	limit  uint = 10
	offset uint = 0
)

var _ Service = (*provisionService)(nil)

// Service specifies Provision service API.
type Service interface {
	// Provision is the only method this API specifies. Depending on the configuration,
	// the following actions will can be executed:
	// - create a Client based on external_id (eg. MAC address)
	// - create multiple Channels
	// - create Bootstrap configuration
	// - whitelist Client in Bootstrap configuration == connect Client to Channels
	Provision(ctx context.Context, domainID, token, name, externalID, externalKey string) (Result, error)

	// Mapping returns current configuration used for provision
	// useful for using in ui to create configuration that matches
	// one created with Provision method.
	Mapping(ctx context.Context, token string) (map[string]any, error)

	// Certs creates certificate for clients that communicate over mTLS
	// A duration string is a possibly signed sequence of decimal numbers,
	// each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	Cert(ctx context.Context, domainID, token, clientID, duration string) (string, string, error)
}

type provisionService struct {
	logger *slog.Logger
	sdk    sdk.SDK
	csdk   csdk.SDK
	conf   Config
}

// Result represent what is created with additional info.
type Result struct {
	Clients     []smqSDK.Client   `json:"clients,omitempty"`
	Channels    []smqSDK.Channel  `json:"channels,omitempty"`
	ClientCert  map[string]string `json:"client_cert,omitempty"`
	ClientKey   map[string]string `json:"client_key,omitempty"`
	CACert      string            `json:"ca_cert,omitempty"`
	Whitelisted map[string]bool   `json:"whitelisted,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// New returns new provision service.
func New(cfg Config, mgsdk sdk.SDK, certsSdk csdk.SDK, logger *slog.Logger) Service {
	return &provisionService{
		logger: logger,
		csdk:   certsSdk,
		conf:   cfg,
		sdk:    mgsdk,
	}
}

// Mapping retrieves current configuration.
func (ps *provisionService) Mapping(ctx context.Context, token string) (map[string]any, error) {
	pm := smqSDK.PageMetadata{
		Offset: uint64(offset),
		Limit:  uint64(limit),
	}

	if _, err := ps.sdk.Users(ctx, pm, token); err != nil {
		return map[string]any{}, errors.Wrap(ErrUnauthorized, err)
	}

	return ps.conf.Bootstrap.Content, nil
}

// Provision is provision method for creating setup according to
// provision layout specified in config.toml.
func (ps *provisionService) Provision(ctx context.Context, domainID, token, name, externalID, externalKey string) (res Result, err error) {
	var channels []smqSDK.Channel
	var clients []smqSDK.Client
	defer ps.recover(ctx, &err, &clients, &channels, &domainID, &token)

	token, err = ps.createTokenIfEmpty(ctx, token)
	if err != nil {
		return res, errors.Wrap(ErrFailedToCreateToken, err)
	}

	if len(ps.conf.Clients) == 0 {
		return res, ErrEmptyClientsList
	}
	if len(ps.conf.Channels) == 0 {
		return res, ErrEmptyChannelsList
	}
	for _, c := range ps.conf.Clients {
		// If client in configs contains metadata with external_id
		// set value for it from the provision request
		if _, ok := c.Metadata[externalIDKey]; ok {
			c.Metadata[externalIDKey] = externalID
		}

		cli := smqSDK.Client{
			Metadata: c.Metadata,
		}
		if name == "" {
			name = c.Name
		}
		cli.Name = name
		cli, err := ps.sdk.CreateClient(ctx, cli, domainID, token)
		if err != nil {
			res.Error = err.Error()
			return res, errors.Wrap(ErrFailedClientCreation, err)
		}

		// Get newly created client (in order to get the key).
		cli, err = ps.sdk.Client(ctx, cli.ID, domainID, token)
		if err != nil {
			e := errors.Wrap(err, fmt.Errorf("client id: %s", cli.ID))
			return res, errors.Wrap(ErrFailedClientRetrieval, e)
		}
		clients = append(clients, cli)
	}

	for _, channel := range ps.conf.Channels {
		ch := smqSDK.Channel{
			Name:     name + "_" + channel.Name,
			Metadata: smqSDK.Metadata(channel.Metadata),
		}
		ch, err := ps.sdk.CreateChannel(ctx, ch, domainID, token)
		if err != nil {
			return res, errors.Wrap(ErrFailedChannelCreation, err)
		}
		ch, err = ps.sdk.Channel(ctx, ch.ID, domainID, token)
		if err != nil {
			e := errors.Wrap(err, fmt.Errorf("channel id: %s", ch.ID))
			return res, errors.Wrap(ErrFailedChannelRetrieval, e)
		}
		channels = append(channels, ch)
	}

	res = Result{
		Clients:     clients,
		Channels:    channels,
		Whitelisted: map[string]bool{},
		ClientCert:  map[string]string{},
		ClientKey:   map[string]string{},
	}

	var cert certs.Certificate
	var bsConfig sdk.BootstrapConfig
	for _, c := range clients {
		var chanIDs []string

		for _, ch := range channels {
			chanIDs = append(chanIDs, ch.ID)
		}
		content, err := json.Marshal(ps.conf.Bootstrap.Content)
		if err != nil {
			return Result{}, errors.Wrap(ErrFailedBootstrap, err)
		}

		if ps.conf.Bootstrap.Provision && needsBootstrap(c) {
			bsReq := sdk.BootstrapConfig{
				ClientID:    c.ID,
				ExternalID:  externalID,
				ExternalKey: externalKey,
				Channels:    chanIDs,
				CACert:      res.CACert,
				ClientCert:  string(cert.Certificate),
				ClientKey:   string(cert.Key),
				Content:     string(content),
			}
			bsid, err := ps.sdk.AddBootstrap(ctx, bsReq, domainID, token)
			if err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrap, err)
			}

			bsConfig, err = ps.sdk.ViewBootstrap(ctx, bsid, domainID, token)
			if err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrapValidate, err)
			}
		}

		if ps.conf.Bootstrap.X509Provision {
			var cert csdk.Certificate

			cert, err = ps.csdk.IssueCert(ctx, c.ID, ps.conf.Cert.TTL, nil, csdk.Options{}, domainID, token)
			if err != nil {
				e := errors.Wrap(err, fmt.Errorf("client id: %s", c.ID))
				return res, errors.Wrap(ErrFailedCertCreation, e)
			}
			cert, err := ps.csdk.ViewCert(ctx, cert.SerialNumber, domainID, token)
			if err != nil {
				return res, errors.Wrap(ErrFailedCertView, err)
			}

			res.ClientCert[c.ID] = cert.Certificate
			res.ClientKey[c.ID] = cert.Key
			res.CACert = ""

			if needsBootstrap(c) {
				if _, err = ps.sdk.UpdateBootstrapCerts(ctx, bsConfig.ClientID, cert.Certificate, cert.Key, "", domainID, token); err != nil {
					return Result{}, errors.Wrap(ErrFailedCertCreation, err)
				}
			}
		}

		if ps.conf.Bootstrap.AutoWhiteList {
			if err := ps.sdk.Whitelist(ctx, c.ID, Active, domainID, token); err != nil {
				res.Error = err.Error()
				return res, ErrClientUpdate
			}
			res.Whitelisted[c.ID] = true
		}
	}

	if err = ps.updateGateway(ctx, domainID, token, bsConfig, channels); err != nil {
		return res, err
	}
	return res, nil
}

func (ps *provisionService) Cert(ctx context.Context, domainID, token, clientID, ttl string) (string, string, error) {
	token, err := ps.createTokenIfEmpty(ctx, token)
	if err != nil {
		return "", "", errors.Wrap(ErrFailedToCreateToken, err)
	}

	c, err := ps.sdk.Client(ctx, clientID, domainID, token)
	if err != nil {
		return "", "", errors.Wrap(ErrUnauthorized, err)
	}
	cert, err := ps.csdk.IssueCert(ctx, c.ID, ps.conf.Cert.TTL, []string{}, csdk.Options{}, domainID, token)
	if err != nil {
		return "", "", errors.Wrap(ErrFailedCertCreation, err)
	}
	cert, err = ps.csdk.ViewCert(ctx, cert.SerialNumber, domainID, token)
	if err != nil {
		return "", "", errors.Wrap(ErrFailedCertView, err)
	}
	return cert.Certificate, cert.Key, err
}

func (ps *provisionService) createTokenIfEmpty(ctx context.Context, token string) (string, error) {
	if token != "" {
		return token, nil
	}

	// If no token in request is provided
	// use API key provided in config file or env
	if ps.conf.Server.MgAPIKey != "" {
		return ps.conf.Server.MgAPIKey, nil
	}

	// If no API key use username and password provided to create access token.
	if ps.conf.Server.MgUsername == "" || ps.conf.Server.MgPass == "" {
		return token, ErrMissingCredentials
	}

	u := smqSDK.Login{
		Username: ps.conf.Server.MgUsername,
		Password: ps.conf.Server.MgPass,
	}
	tkn, err := ps.sdk.CreateToken(ctx, u)
	if err != nil {
		return token, errors.Wrap(ErrFailedToCreateToken, err)
	}

	return tkn.AccessToken, nil
}

func (ps *provisionService) updateGateway(ctx context.Context, domainID, token string, bs sdk.BootstrapConfig, channels []smqSDK.Channel) error {
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
	gw.CfgID = bs.ClientID
	gw.Type = gateway

	c, sdkerr := ps.sdk.Client(ctx, bs.ClientID, domainID, token)
	if sdkerr != nil {
		return errors.Wrap(ErrGatewayUpdate, sdkerr)
	}
	b, err := json.Marshal(gw)
	if err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	if err := json.Unmarshal(b, &c.Metadata); err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	if _, err := ps.sdk.UpdateClient(ctx, c, domainID, token); err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	return nil
}

func (ps *provisionService) errLog(err error) {
	if err != nil {
		ps.logger.Error(fmt.Sprintf("Error recovering: %s", err))
	}
}

func clean(ctx context.Context, ps *provisionService, clients []smqSDK.Client, channels []smqSDK.Channel, domainID, token string) {
	for _, t := range clients {
		err := ps.sdk.DeleteClient(ctx, t.ID, domainID, token)
		ps.errLog(err)
	}
	for _, c := range channels {
		err := ps.sdk.DeleteChannel(ctx, c.ID, domainID, token)
		ps.errLog(err)
	}
}

func (ps *provisionService) recover(ctx context.Context, e *error, ths *[]smqSDK.Client, chs *[]smqSDK.Channel, dm, tkn *string) {
	if e == nil {
		return
	}
	clients, channels, domainID, token, err := *ths, *chs, *dm, *tkn, *e

	if errors.Contains(err, ErrFailedClientRetrieval) || errors.Contains(err, ErrFailedChannelCreation) {
		for _, c := range clients {
			err := ps.sdk.DeleteClient(ctx, c.ID, domainID, token)
			ps.errLog(err)
		}
		return
	}

	if errors.Contains(err, ErrFailedBootstrap) || errors.Contains(err, ErrFailedChannelRetrieval) {
		clean(ctx, ps, clients, channels, domainID, token)
		return
	}

	if errors.Contains(err, ErrFailedBootstrapValidate) || errors.Contains(err, ErrFailedCertCreation) {
		clean(ctx, ps, clients, channels, domainID, token)
		for _, c := range clients {
			if needsBootstrap(c) {
				ps.errLog(ps.sdk.RemoveBootstrap(ctx, c.ID, domainID, token))
			}
		}
		return
	}

	if errors.Contains(err, ErrFailedBootstrapValidate) || errors.Contains(err, ErrFailedCertCreation) {
		clean(ctx, ps, clients, channels, domainID, token)
		for _, c := range clients {
			if needsBootstrap(c) {
				bs, err := ps.sdk.ViewBootstrap(ctx, c.ID, domainID, token)
				ps.errLog(errors.Wrap(ErrFailedBootstrapRetrieval, err))
				ps.errLog(ps.sdk.RemoveBootstrap(ctx, bs.ClientID, domainID, token))
			}
		}
	}

	if errors.Contains(err, ErrClientUpdate) || errors.Contains(err, ErrGatewayUpdate) {
		clean(ctx, ps, clients, channels, domainID, token)
		for _, c := range clients {
			if ps.conf.Bootstrap.X509Provision && needsBootstrap(c) {
				err := ps.csdk.RevokeCert(ctx, c.ID, domainID, token)
				ps.errLog(err)
			}
			if needsBootstrap(c) {
				bs, err := ps.sdk.ViewBootstrap(ctx, c.ID, domainID, token)
				ps.errLog(errors.Wrap(ErrFailedBootstrapRetrieval, err))
				ps.errLog(ps.sdk.RemoveBootstrap(ctx, bs.ClientID, domainID, token))
			}
		}
		return
	}
}

func needsBootstrap(c smqSDK.Client) bool {
	if c.Metadata == nil {
		return false
	}

	if _, ok := c.Metadata[externalIDKey]; ok {
		return true
	}
	return false
}
