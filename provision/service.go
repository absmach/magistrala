// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/sdk"
	smqSDK "github.com/absmach/magistrala/pkg/sdk"
)

const (
	externalIDKey = "external_id"
	gateway       = "gateway"

	control = "control"
	data    = "data"
	export  = "export"
)

var (
	ErrUnauthorized             = errors.NewAuthNError("unauthorized access")
	ErrFailedToCreateToken      = errors.NewAuthNError("failed to create access token")
	ErrEmptyClientsList         = errors.NewRequestError("clients list in configuration empty")
	ErrClientUpdate             = errors.NewRequestError("failed to update client")
	ErrEmptyChannelsList        = errors.NewRequestError("channels list in configuration is empty")
	ErrFailedChannelCreation    = errors.NewRequestError("failed to create channel")
	ErrFailedChannelRetrieval   = errors.NewRequestError("failed to retrieve channel")
	ErrFailedClientCreation     = errors.NewRequestError("failed to create client")
	ErrFailedClientRetrieval    = errors.NewRequestError("failed to retrieve client")
	ErrMissingCredentials       = errors.NewRequestError("missing credentials")
	ErrFailedBootstrapRetrieval = errors.NewServiceError("failed to retrieve bootstrap")
	ErrFailedCertCreation       = errors.NewServiceError("failed to create certificates")
	ErrFailedCertView           = errors.NewServiceError("failed to view certificate")
	ErrFailedBootstrap          = errors.NewServiceError("failed to create bootstrap config")
	ErrFailedBootstrapValidate  = errors.NewServiceError("failed to validate bootstrap config creation")
	ErrFailedBootstrapBinding   = errors.NewServiceError("failed to bind bootstrap resources")
	ErrGatewayUpdate            = errors.NewServiceError("failed to update gateway metadata")
)

var _ Service = (*provisionService)(nil)

// Service specifies Provision service API.
type Service interface {
	// Provision is the only method this API specifies. Depending on the configuration,
	// the following actions will can be executed:
	// - create a Client based on external_id (eg. MAC address)
	// - create multiple Channels
	// - create Bootstrap configuration
	// - enable created Bootstrap enrollments
	Provision(ctx context.Context, domainID, token, name, externalID, externalKey string) (Result, error)

	// Mapping returns current configuration used for provision
	// useful for using in ui to create configuration that matches
	// one created with Provision method.
	Mapping() map[string]any

	// Certs creates certificate for clients that communicate over mTLS
	// A duration string is a possibly signed sequence of decimal numbers,
	// each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	Cert(ctx context.Context, domainID, token, clientID, duration string) (string, string, error)
}

type provisionService struct {
	logger *slog.Logger
	sdk    sdk.SDK
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
func New(cfg Config, mgsdk sdk.SDK, logger *slog.Logger) Service {
	return &provisionService{
		logger: logger,
		conf:   cfg,
		sdk:    mgsdk,
	}
}

// Mapping retrieves current configuration.
func (ps *provisionService) Mapping() map[string]any {
	return ps.conf.Bootstrap.Content
}

// Provision is provision method for creating setup according to
// provision layout specified in config.toml.
func (ps *provisionService) Provision(ctx context.Context, domainID, token, name, externalID, externalKey string) (res Result, err error) {
	var channels []smqSDK.Channel
	var clients []smqSDK.Client
	var bootstrapIDs []string
	defer ps.recover(ctx, &err, &clients, &channels, &bootstrapIDs, domainID, token)

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

	content, err := json.Marshal(ps.conf.Bootstrap.Content)
	if err != nil {
		return Result{}, errors.Wrap(ErrFailedBootstrap, err)
	}

	bootstrapConfigs := make(map[string]sdk.BootstrapConfig)
	var gatewayConfig sdk.BootstrapConfig
	var gatewayClientID string
	for _, c := range clients {
		if ps.conf.Bootstrap.Provision && needsBootstrap(c) {
			bsReq := sdk.BootstrapConfig{
				ExternalID:    externalID,
				ExternalKey:   externalKey,
				Name:          name,
				CACert:        res.CACert,
				ClientCert:    "",
				ClientKey:     "",
				Content:       string(content),
				ProfileID:     ps.conf.Bootstrap.ProfileID,
				RenderContext: ps.bootstrapRenderContext(externalID, name),
			}
			bsid, err := ps.sdk.AddBootstrap(ctx, bsReq, domainID, token)
			if err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrap, err)
			}
			bootstrapIDs = append(bootstrapIDs, bsid)

			bsConfig, err := ps.sdk.ViewBootstrap(ctx, bsid, domainID, token)
			if err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrapValidate, err)
			}
			bootstrapConfigs[c.ID] = bsConfig
			gatewayConfig = bsConfig
			gatewayClientID = c.ID

			if err := ps.bindBootstrapResources(ctx, bsConfig.ID, c, clients, channels, domainID, token); err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrapBinding, err)
			}
		}

		if ps.conf.Bootstrap.X509Provision {
			var cert smqSDK.Certificate

			cert, err = ps.sdk.IssueCert(ctx, c.ID, ps.conf.Cert.TTL, nil, smqSDK.Options{}, domainID, token)
			if err != nil {
				e := errors.Wrap(err, fmt.Errorf("client id: %s", c.ID))
				return res, errors.Wrap(ErrFailedCertCreation, e)
			}
			cert, err := ps.sdk.ViewCert(ctx, cert.SerialNumber, domainID, token)
			if err != nil {
				return res, errors.Wrap(ErrFailedCertView, err)
			}

			res.ClientCert[c.ID] = cert.Certificate
			res.ClientKey[c.ID] = cert.Key
			res.CACert = ""

			if bsConfig, ok := bootstrapConfigs[c.ID]; ok {
				updated, err := ps.sdk.UpdateBootstrapCerts(ctx, bsConfig.ID, cert.Certificate, cert.Key, "", domainID, token)
				if err != nil {
					return Result{}, errors.Wrap(ErrFailedCertCreation, err)
				}
				bootstrapConfigs[c.ID] = updated
				if gatewayClientID == c.ID {
					gatewayConfig = updated
				}
			}
		}

		if ps.conf.Bootstrap.AutoWhiteList {
			if bsConfig, ok := bootstrapConfigs[c.ID]; ok {
				if err := ps.sdk.Whitelist(ctx, bsConfig.ID, smqSDK.BootstrapEnabledStatus, domainID, token); err != nil {
					res.Error = err.Error()
					return res, ErrClientUpdate
				}
				res.Whitelisted[bsConfig.ID] = true
			}
		}
	}

	if gatewayClientID != "" && gatewayConfig.ID != "" {
		if err = ps.updateGateway(ctx, domainID, token, gatewayClientID, gatewayConfig, externalKey, channels); err != nil {
			return res, err
		}
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
	cert, err := ps.sdk.IssueCert(ctx, c.ID, ps.conf.Cert.TTL, []string{}, smqSDK.Options{}, domainID, token)
	if err != nil {
		return "", "", errors.Wrap(ErrFailedCertCreation, err)
	}
	cert, err = ps.sdk.ViewCert(ctx, cert.SerialNumber, domainID, token)
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

func (ps *provisionService) updateGateway(ctx context.Context, domainID, token, gatewayClientID string, bs sdk.BootstrapConfig, externalKey string, channels []smqSDK.Channel) error {
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
	gw.ExternalKey = externalKey
	gw.CfgID = bs.ID
	gw.Type = gateway

	c, sdkerr := ps.sdk.Client(ctx, gatewayClientID, domainID, token)
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

func (ps *provisionService) bootstrapRenderContext(externalID, name string) map[string]any {
	renderContext := make(map[string]any, len(ps.conf.Bootstrap.RenderContext)+2)
	for k, v := range ps.conf.Bootstrap.RenderContext {
		renderContext[k] = v
	}
	if externalID != "" {
		renderContext[externalIDKey] = externalID
	}
	if name != "" {
		renderContext["name"] = name
	}
	if len(renderContext) == 0 {
		return nil
	}
	return renderContext
}

func (ps *provisionService) bindBootstrapResources(ctx context.Context, configID string, bootstrapClient smqSDK.Client, clients []smqSDK.Client, channels []smqSDK.Channel, domainID, token string) error {
	if len(ps.conf.Bootstrap.Bindings) == 0 {
		return nil
	}

	requests := make([]smqSDK.BootstrapBindingRequest, 0, len(ps.conf.Bootstrap.Bindings))
	for _, binding := range ps.conf.Bootstrap.Bindings {
		resourceID := ps.bindingResourceID(binding, bootstrapClient, clients, channels)
		if resourceID == "" {
			return fmt.Errorf("resource for bootstrap binding slot %q not found", binding.Slot)
		}
		requests = append(requests, smqSDK.BootstrapBindingRequest{
			Slot:       binding.Slot,
			Type:       binding.Type,
			ResourceID: resourceID,
		})
	}

	return ps.sdk.BindBootstrapResources(ctx, configID, requests, domainID, token)
}

func (ps *provisionService) bindingResourceID(binding BootstrapBinding, bootstrapClient smqSDK.Client, clients []smqSDK.Client, channels []smqSDK.Channel) string {
	switch binding.Type {
	case "client":
		if matchesClientBinding(binding, bootstrapClient) {
			return bootstrapClient.ID
		}
		for _, client := range clients {
			if matchesClientBinding(binding, client) {
				return client.ID
			}
		}
	case "channel":
		for _, channel := range channels {
			if matchesChannelBinding(binding, channel) {
				return channel.ID
			}
		}
	}
	return ""
}

func matchesClientBinding(binding BootstrapBinding, client smqSDK.Client) bool {
	if binding.Name != "" && client.Name != binding.Name {
		return false
	}
	if binding.MetadataKey != "" {
		return metadataValue(client.Metadata, binding.MetadataKey) == binding.MetadataValue
	}
	return binding.Name != "" || client.ID != ""
}

func matchesChannelBinding(binding BootstrapBinding, channel smqSDK.Channel) bool {
	if binding.Name != "" && channel.Name != binding.Name {
		return false
	}
	if binding.MetadataKey != "" {
		return metadataValue(channel.Metadata, binding.MetadataKey) == binding.MetadataValue
	}
	return binding.Name != "" || channel.ID != ""
}

func metadataValue(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key]; ok {
		return fmt.Sprint(value)
	}
	return ""
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

func (ps *provisionService) removeBootstraps(ctx context.Context, ids []string, domainID, token string) {
	for _, id := range ids {
		ps.errLog(ps.sdk.RemoveBootstrap(ctx, id, domainID, token))
	}
}

func (ps *provisionService) recover(ctx context.Context, e *error, ths *[]smqSDK.Client, chs *[]smqSDK.Channel, bootstrapIDs *[]string, domainID, token string) {
	if e == nil {
		return
	}
	clients, channels, bootstraps, err := *ths, *chs, *bootstrapIDs, *e

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

	if errors.Contains(err, ErrFailedBootstrapValidate) || errors.Contains(err, ErrFailedCertCreation) || errors.Contains(err, ErrFailedBootstrapBinding) {
		clean(ctx, ps, clients, channels, domainID, token)
		ps.removeBootstraps(ctx, bootstraps, domainID, token)
		return
	}

	if errors.Contains(err, ErrClientUpdate) || errors.Contains(err, ErrGatewayUpdate) {
		clean(ctx, ps, clients, channels, domainID, token)
		for _, c := range clients {
			if ps.conf.Bootstrap.X509Provision && needsBootstrap(c) {
				err := ps.sdk.RevokeCert(ctx, c.ID, domainID, token)
				ps.errLog(err)
			}
		}
		ps.removeBootstraps(ctx, bootstraps, domainID, token)
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
