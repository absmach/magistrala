// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
)

const configsEndpoint = "configs"
const bootstrapEndpoint = "bootstrap"
const whitelistEndpoint = "state"
const bootstrapCertsEndpoint = "things/configs/certs"

// BootstrapConfig represents Configuration entity. It wraps information about external entity
// as well as info about corresponding Mainflux entities.
// MFThing represents corresponding Mainflux Thing ID.
// MFKey is key of corresponding Mainflux Thing.
// MFChannels is a list of Mainflux Channels corresponding Mainflux Thing connects to.
type BootstrapConfig struct {
	ThingID     string    `json:"thing_id,omitempty"`
	Channels    []string  `json:"channels,omitempty"`
	ExternalID  string    `json:"external_id,omitempty"`
	ExternalKey string    `json:"external_key,omitempty"`
	MFThing     string    `json:"mainflux_id,omitempty"`
	MFChannels  []Channel `json:"mainflux_channels,omitempty"`
	MFKey       string    `json:"mainflux_key,omitempty"`
	Name        string    `json:"name,omitempty"`
	ClientCert  string    `json:"client_cert,omitempty"`
	ClientKey   string    `json:"client_key,omitempty"`
	CACert      string    `json:"ca_cert,omitempty"`
	Content     string    `json:"content,omitempty"`
	State       int       `json:"state,omitempty"`
}

type ConfigUpdateCertReq struct {
	ClientCert string `json:"client_cert"`
	ClientKey  string `json:"client_key"`
	CACert     string `json:"ca_cert"`
}

func (sdk mfSDK) AddBootstrap(cfg BootstrapConfig, token string) (string, errors.SDKError) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.bootstrapURL, configsEndpoint)

	headers, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusOK, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	id := strings.TrimPrefix(headers.Get("Location"), "/things/configs/")
	return id, nil
}

func (sdk mfSDK) Whitelist(cfg BootstrapConfig, token string) errors.SDKError {
	data, err := json.Marshal(BootstrapConfig{State: cfg.State})
	if err != nil {
		return errors.NewSDKError(err)
	}

	if cfg.MFThing == "" {
		return errors.NewSDKError(errors.ErrNotFoundParam)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, whitelistEndpoint, cfg.MFThing)

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, string(CTJSON), data, http.StatusCreated, http.StatusOK)
	return sdkerr
}

func (sdk mfSDK) ViewBootstrap(id, token string) (BootstrapConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, configsEndpoint, id)
	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mfSDK) UpdateBootstrap(cfg BootstrapConfig, token string) errors.SDKError {
	data, err := json.Marshal(cfg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, configsEndpoint, cfg.MFThing)
	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, string(CTJSON), data, http.StatusOK)
	return sdkerr
}
func (sdk mfSDK) UpdateBootstrapCerts(id, clientCert, clientKey, ca, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, bootstrapCertsEndpoint, id)
	request := ConfigUpdateCertReq{
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     ca,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return errors.NewSDKError(err)
	}


	_, _, sdkerr := sdk.processRequest(http.MethodPatch, url, token, string(CTJSON), data, http.StatusOK)
	return sdkerr
}

func (sdk mfSDK) RemoveBootstrap(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, configsEndpoint, id)
	_, _, err := sdk.processRequest(http.MethodDelete, url, token, string(CTJSON), nil, http.StatusNoContent)
	return err
}

func (sdk mfSDK) Bootstrap(externalID, externalKey string) (BootstrapConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, bootstrapEndpoint, externalID)
	_, body, err := sdk.processRequest(http.MethodGet, url, apiutil.ThingPrefix+externalKey, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}
