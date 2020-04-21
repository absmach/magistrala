package provision

import (
	"fmt"

	"github.com/mainflux/mainflux/logger"
	provsdk "github.com/mainflux/mainflux/provision/sdk"
)

var _ Service = (*provisionService)(nil)

// Service specifies Provision service API.
type Service interface {
	// Provision is the only method this API specifies. Depending on the configuration,
	// the following actions will can be executed:
	// - create a Thing based od mac address
	// - create multiple Channels
	// - connect predefined Things to the created Channels
	// - create Bootstrap configuration
	// - whitelist Thing in Bootstrap configuration == connect Thing to Channels
	Provision(externalID, externalKey string) (Result, error)
}

type provisionService struct {
	logger           logger.Logger
	sdk              provsdk.SDK
	mfEmail          string
	mfPass           string
	x509Provision    bool
	bsProvision      bool
	autoWhiteList    bool
	bsContent        string
	predefinedThings []string
}

// Result represent what is created with additional info.
type Result struct {
	Thing      provsdk.Thing     `json:"thing,omitempty"`
	ThingsID   []string          `json:"thing_ids,omitempty"`
	Channels   []provsdk.Channel `json:"channels,omitempty"`
	ClientCert string            `json:"client_cert,omitempty"`
	ClientKey  string            `json:"client_key,omitempty"`
	CACert     string            `json:"ca_cert,omitempty"`
	Witelisted bool              `json:"whitelisted,omitempty"`
}

// Config represents service config.
type Config struct {
	SDK              provsdk.SDK
	MFEmail          string
	MFPass           string
	X509Provision    bool
	BSProvision      bool
	AutoWhiteList    bool
	BSContent        string
	PredefinedThings []string
}

// New returns new provision service.
func New(cfg Config, logger logger.Logger) Service {
	return &provisionService{
		logger:           logger,
		sdk:              cfg.SDK,
		mfEmail:          cfg.MFEmail,
		mfPass:           cfg.MFPass,
		bsContent:        cfg.BSContent,
		x509Provision:    cfg.X509Provision,
		bsProvision:      cfg.BSProvision,
		autoWhiteList:    cfg.AutoWhiteList,
		predefinedThings: cfg.PredefinedThings,
	}
}

// Provision is provision method for adding devices to proxy.
func (ps *provisionService) Provision(externalID, externalKey string) (res Result, err error) {
	var newThingID, ctrlChanID, dataChanID, token string
	defer ps.recover(&err, &newThingID, &ctrlChanID, &dataChanID, &token)
	channels := make([]string, 0)

	token, err = ps.sdk.CreateToken(ps.mfEmail, ps.mfPass)
	if err != nil {
		return res, err
	}

	newThingID, err = ps.sdk.CreateThing(externalID, "", token)
	if err != nil {
		return res, err
	}

	// Get newly created thing (in order to get the key).
	thingCreated, err := ps.sdk.Thing(newThingID, token)
	if err != nil {
		return res, provsdk.ErrGetThing
	}

	ctrlChannel, err := ps.sdk.CreateChannel("ctrlchan", "control", token)
	if err != nil {
		return res, provsdk.ErrCreateCtrl
	}
	ctrlChanID = ctrlChannel.ID

	dataChannel, err := ps.sdk.CreateChannel("datachan", "data", token)
	if err != nil {
		return res, provsdk.ErrCreateData
	}
	dataChanID = dataChannel.ID

	channels = append(channels, ctrlChanID, dataChanID)

	for _, t := range ps.predefinedThings {
		// Connect predefined Things to control channel.
		err = ps.sdk.Connect(t, ctrlChanID, token)
		if err != nil {
			return res, provsdk.ErrConn
		}
	}

	res = Result{
		Thing:    thingCreated,
		Channels: []provsdk.Channel{ctrlChannel, dataChannel},
	}

	if ps.x509Provision {
		certs, err := ps.sdk.Cert(thingCreated.ID, thingCreated.Key, token)
		if err != nil {
			return res, provsdk.ErrCerts
		}
		res.ClientCert = certs.ClientCert
		res.ClientKey = certs.ClientKey
		res.CACert = certs.CACert
	}

	if ps.bsProvision {
		bsReq := provsdk.BSConfig{
			ThingID:     thingCreated.ID,
			ExternalID:  externalID,
			ExternalKey: externalKey,
			Channels:    channels,
			CACert:      res.CACert,
			ClientCert:  res.ClientCert,
			ClientKey:   res.ClientKey,
			Content:     ps.bsContent,
		}

		if err := ps.sdk.SaveConfig(bsReq, token); err != nil {
			return Result{}, provsdk.ErrConfig
		}
	}

	if ps.autoWhiteList {
		wlReq := map[string]int{
			"state": 1,
		}
		if err := ps.sdk.Whitelist(thingCreated.ID, wlReq, token); err != nil {
			return res, provsdk.ErrWhitelist
		}
		res.Witelisted = true
	}

	return res, nil
}

func (ps *provisionService) errLog(err error) {
	if err != nil {
		ps.logger.Error(fmt.Sprintf("Error recovering: %s", err))
	}
}

func clean(ps *provisionService, thingID, ctrlChan, dataChan, token string) {
	ps.errLog(ps.sdk.DeleteThing(thingID, token))
	ps.errLog(ps.sdk.DeleteChannel(ctrlChan, token))
	ps.errLog(ps.sdk.DeleteChannel(dataChan, token))
}

func (ps *provisionService) recover(err *error, thingID, ctrlChan, dataChan, token *string) {
	thing, ctrl, data, tkn, e := *thingID, *ctrlChan, *dataChan, *token, *err
	switch e {
	case nil:
		return
	case provsdk.ErrGetThing, provsdk.ErrCreateCtrl:
		ps.errLog(ps.sdk.DeleteThing(thing, tkn))
	case provsdk.ErrCreateData:
		ps.errLog(ps.sdk.DeleteThing(thing, tkn))
		ps.errLog(ps.sdk.DeleteChannel(ctrl, tkn))
	case provsdk.ErrConn, provsdk.ErrCerts:
		clean(ps, thing, ctrl, data, tkn)
	case provsdk.ErrConfig:
		clean(ps, thing, ctrl, data, tkn)
		if ps.x509Provision {
			ps.errLog(ps.sdk.RemoveCert(thing, tkn))
		}
	case provsdk.ErrWhitelist:
		clean(ps, thing, ctrl, data, tkn)
		if ps.x509Provision {
			ps.errLog(ps.sdk.RemoveCert(thing, tkn))
		}
		ps.errLog(ps.sdk.RemoveConfig(thing, tkn))
	}
}
