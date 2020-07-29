// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
)

const configsEndpoint = "configs"
const bootstrapEndpoint = "bootstrap"
const whitelistEndpoint = "state"
const bootstrapCertsEndpoint = "configs/certs"

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

func (sdk mfSDK) AddBootstrap(token string, cfg BootstrapConfig) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, configsEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), "/things/configs/")
	return id, nil
}

func (sdk mfSDK) Whitelist(token string, cfg BootstrapConfig) error {
	data, err := json.Marshal(BootstrapConfig{State: cfg.State})
	if err != nil {
		return errors.Wrap(ErrFailedWhitelist, err)
	}

	if cfg.MFThing == "" {
		return ErrFailedWhitelist
	}

	endpoint := fmt.Sprintf("%s/%s", whitelistEndpoint, cfg.MFThing)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(ErrFailedWhitelist, err)
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return errors.Wrap(ErrFailedWhitelist, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedWhitelist, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) ViewBootstrap(token, id string) (BootstrapConfig, error) {
	endpoint := fmt.Sprintf("%s/%s", configsEndpoint, id)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return BootstrapConfig{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return BootstrapConfig{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BootstrapConfig{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return BootstrapConfig{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, err
	}

	return bc, nil
}

func (sdk mfSDK) UpdateBootstrap(token string, cfg BootstrapConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", configsEndpoint, cfg.MFThing)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
	}

	return nil
}
func (sdk mfSDK) UpdateBootstrapCerts(token, id, clientCert, clientKey, ca string) error {
	endpoint := fmt.Sprintf("%s/%s", bootstrapCertsEndpoint, id)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)
	request := ConfigUpdateCertReq{
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     ca,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(ErrFailedCertUpdate, err)
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedCertUpdate, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) RemoveBootstrap(token, id string) error {
	endpoint := fmt.Sprintf("%s/%s", configsEndpoint, id)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Bootstrap(externalKey, externalID string) (BootstrapConfig, error) {
	endpoint := fmt.Sprintf("%s/%s", bootstrapEndpoint, externalID)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return BootstrapConfig{}, err
	}

	resp, err := sdk.sendRequest(req, externalKey, string(CTJSON))
	if err != nil {
		return BootstrapConfig{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BootstrapConfig{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return BootstrapConfig{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, err
	}

	return bc, nil
}
