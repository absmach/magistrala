package provision

import (
	"encoding/json"
	"fmt"

	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/logger"
	SDK "github.com/mainflux/mainflux/sdk/go"
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
	ErrFailedToCreateToken      = errors.New("failed to create access token")
	ErrEmptyThingsList          = errors.New("things list in configuration empty")
	ErrEmptyChannelsList        = errors.New("channels list in configuration is empty")
	ErrFailedChannelCreation    = errors.New("failed to create channel")
	ErrFailedChannelRetrieval   = errors.New("failed to retrieve channel")
	ErrFailedThingCreation      = errors.New("failed to create thing")
	ErrFailedThingRetrieval     = errors.New("failed to retrieve thing")
	ErrMissingCredentials       = errors.New("missing credentials")
	ErrFailedBootstrapRetrieval = errors.New("failed to retrieve bootstrap")
	ErrFailedCertCreation       = errors.New("failed to create certificates")
	ErrFailedBootstrap          = errors.New("failed to create bootstrap config")
	ErrGatewayUpdate            = errors.New("failed to updated gateway metadata")
)

var _ Service = (*provisionService)(nil)

// Service specifies Provision service API.
type Service interface {
	// Provision is the only method this API specifies. Depending on the configuration,
	// the following actions will can be executed:
	// - create a Thing based on external_id (eg. MAC address)
	// - create multiple Channels
	// - create Bootstrap configuration
	// - whitelist Thing in Bootstrap configuration == connect Thing to Channels
	Provision(name, token, externalID, externalKey string) (Result, error)
}

type provisionService struct {
	logger logger.Logger
	sdk    SDK.SDK
	conf   Config
}

// Result represent what is created with additional info.
type Result struct {
	Things      []SDK.Thing       `json:"things,omitempty"`
	Channels    []SDK.Channel     `json:"channels,omitempty"`
	ClientCert  map[string]string `json:"client_cert,omitempty"`
	ClientKey   map[string]string `json:"client_key,omitempty"`
	CACert      string            `json:"ca_cert,omitempty"`
	Whitelisted map[string]bool   `json:"whitelisted,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// New returns new provision service.
func New(cfg Config, sdk SDK.SDK, logger logger.Logger) Service {
	return &provisionService{
		logger: logger,
		conf:   cfg,
		sdk:    sdk,
	}
}

// Provision is provision method for creating setup according to
// provision layout specified in config.toml
func (ps *provisionService) Provision(name, token, externalID, externalKey string) (res Result, err error) {
	var channels []SDK.Channel
	var things []SDK.Thing
	defer ps.recover(&err, &things, &channels, &token)

	if token == "" {
		token = ps.conf.Server.MfAPIKey
		if token == "" {
			if ps.conf.Server.MfUser == "" || ps.conf.Server.MfPass == "" {
				return res, ErrMissingCredentials
			}
			u := SDK.User{
				Email:    ps.conf.Server.MfUser,
				Password: ps.conf.Server.MfPass,
			}
			token, err = ps.sdk.CreateToken(u)
			if err != nil {
				return res, errors.Wrap(ErrFailedToCreateToken, err)
			}
		}
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

		th := SDK.Thing{
			Metadata: thing.Metadata,
		}
		if name == "" {
			name = thing.Name
		}
		th.Name = name
		thID, err := ps.sdk.CreateThing(th, token)
		if err != nil {
			res.Error = err.Error()
			return res, errors.Wrap(ErrFailedThingCreation, err)
		}
		// Get newly created thing (in order to get the key).
		th, err = ps.sdk.Thing(thID, token)
		if err != nil {
			e := errors.Wrap(err, fmt.Errorf("thing id: %s", thID))
			return res, errors.Wrap(ErrFailedThingRetrieval, e)
		}
		things = append(things, th)
	}

	for _, channel := range ps.conf.Channels {
		ch := SDK.Channel{
			Name:     channel.Name,
			Metadata: channel.Metadata,
		}
		chCreated, err := ps.sdk.CreateChannel(ch, token)
		if err != nil {
			return res, err
		}
		ch, err = ps.sdk.Channel(chCreated, token)
		if err != nil {
			e := errors.Wrap(err, fmt.Errorf("channel id: %s", chCreated))
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

	var cert SDK.Cert
	var bs SDK.BootstrapConfig
	for _, thing := range things {
		bootstrap := false
		if _, ok := thing.Metadata[externalIDKey]; ok {
			bootstrap = true
		}
		var chanIDs []string
		for _, ch := range channels {
			chanIDs = append(chanIDs, ch.ID)
		}
		if ps.conf.Bootstrap.Provision && bootstrap {
			bsReq := SDK.BootstrapConfig{
				ThingID:     thing.ID,
				ExternalID:  externalID,
				ExternalKey: externalKey,
				Channels:    chanIDs,
				CACert:      res.CACert,
				ClientCert:  cert.ClientCert,
				ClientKey:   cert.ClientKey,
				Content:     ps.conf.Bootstrap.Content,
			}
			bsid, err := ps.sdk.AddBootstrap(token, bsReq)
			if err != nil {
				return Result{}, errors.Wrap(ErrFailedBootstrap, err)
			}

			bs, err = ps.sdk.ViewBootstrap(token, bsid)
			if err != nil {
				return Result{}, err
			}
		}

		if ps.conf.Bootstrap.X509Provision {
			cert, err = ps.sdk.Cert(thing.ID, thing.Key, token)
			if err != nil {
				e := errors.Wrap(err, fmt.Errorf("thing id: %s", thing.ID))
				return res, errors.Wrap(ErrFailedCertCreation, e)
			}
			res.ClientCert[thing.ID] = cert.ClientCert
			res.ClientKey[thing.ID] = cert.ClientKey
			res.CACert = cert.CACert
		}

		if ps.conf.Bootstrap.AutoWhiteList {
			wlReq := SDK.BootstrapConfig{
				MFThing: thing.ID,
				State:   Active,
			}
			if err := ps.sdk.Whitelist(token, wlReq); err != nil {
				res.Error = err.Error()
				return res, SDK.ErrFailedWhitelist
			}
			res.Whitelisted[thing.ID] = true
		}

	}

	ps.updateGateway(token, bs, channels)
	return res, nil
}

func (ps *provisionService) updateGateway(token string, bs SDK.BootstrapConfig, channels []SDK.Channel) error {
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
	gw.CfgID = bs.MFThing
	gw.Type = gateway

	th, err := ps.sdk.Thing(bs.MFThing, token)
	if err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	b, err := json.Marshal(gw)
	if err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	if err := json.Unmarshal(b, &th.Metadata); err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	if err := ps.sdk.UpdateThing(th, token); err != nil {
		return errors.Wrap(ErrGatewayUpdate, err)
	}
	return nil
}

func (ps *provisionService) errLog(err error) {
	if err != nil {
		ps.logger.Error(fmt.Sprintf("Error recovering: %s", err))
	}
}

func clean(ps *provisionService, things []SDK.Thing, channels []SDK.Channel, token string) {
	for _, t := range things {
		ps.errLog(ps.sdk.DeleteThing(t.ID, token))
	}
	for _, c := range channels {
		ps.errLog(ps.sdk.DeleteThing(c.ID, token))
	}
}

func (ps *provisionService) recover(e *error, ths *[]SDK.Thing, chs *[]SDK.Channel, tkn *string) {
	things, channels, token, err := *ths, *chs, *tkn, *e
	if e == nil {
		return
	}
	if errors.Contains(err, ErrFailedThingRetrieval) || errors.Contains(err, ErrFailedChannelCreation) {
		for _, th := range things {
			ps.errLog(ps.sdk.DeleteThing(th.ID, token))
		}
		return
	}

	if errors.Contains(err, ErrFailedChannelRetrieval) || errors.Contains(err, ErrFailedCertCreation) {
		for _, th := range things {
			ps.errLog(ps.sdk.DeleteThing(th.ID, token))
		}
		for _, ch := range channels {
			ps.errLog(ps.sdk.DeleteChannel(ch.ID, token))
		}
		return
	}

	if errors.Contains(err, ErrFailedBootstrap) {
		clean(ps, things, channels, token)
		if ps.conf.Bootstrap.X509Provision {
			for _, th := range things {
				ps.errLog(ps.sdk.RemoveCert(th.ID, token))
			}
		}
		return
	}

	if errors.Contains(err, SDK.ErrFailedWhitelist) {
		clean(ps, things, channels, token)
		for _, th := range things {
			if ps.conf.Bootstrap.X509Provision {
				ps.errLog(ps.sdk.RemoveCert(th.ID, token))
			}
			bs, err := ps.sdk.ViewBootstrap(token, th.ID)
			ps.errLog(errors.Wrap(ErrFailedBootstrapRetrieval, err))
			ps.errLog(ps.sdk.RemoveBootstrap(token, bs.MFThing))
		}
		return
	}

	if errors.Contains(err, ErrGatewayUpdate) {
		clean(ps, things, channels, token)
		for _, th := range things {
			if ps.conf.Bootstrap.X509Provision {
				ps.errLog(ps.sdk.RemoveCert(th.ID, token))
			}
			bs, err := ps.sdk.ViewBootstrap(token, th.ID)
			ps.errLog(errors.Wrap(ErrFailedBootstrapRetrieval, err))
			ps.errLog(ps.sdk.RemoveBootstrap(token, bs.MFThing))
		}
		return
	}

}
