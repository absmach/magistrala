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

	"github.com/mainflux/mainflux/errors"
)

const configsEndpoint = "configs"
const bootstrapEndpoint = "bootstrap"
const whitelistEndpoint = "state"

// BoostrapConfig represents Configuration entity. It wraps information about external entity
// as well as info about corresponding Mainflux entities.
// MFThing represents corresponding Mainflux Thing ID.
// MFKey is key of corresponding Mainflux Thing.
// MFChannels is a list of Mainflux Channels corresponding Mainflux Thing connects to.
type BoostrapConfig struct {
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

func (sdk mfSDK) AddBootstrap(key string, cfg BoostrapConfig) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, configsEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, key, string(CTJSON))
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

func (sdk mfSDK) Whitelist(token string, cfg BoostrapConfig) error {
	data, err := json.Marshal(cfg)
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

func (sdk mfSDK) ViewBoostrap(key, id string) (BoostrapConfig, error) {
	endpoint := fmt.Sprintf("%s/%s", configsEndpoint, id)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return BoostrapConfig{}, err
	}

	resp, err := sdk.sendRequest(req, key, string(CTJSON))
	if err != nil {
		return BoostrapConfig{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BoostrapConfig{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return BoostrapConfig{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var bc BoostrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BoostrapConfig{}, err
	}

	return bc, nil
}

func (sdk mfSDK) UpdateBoostrap(key string, cfg BoostrapConfig) error {
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

	resp, err := sdk.sendRequest(req, key, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) RemoveBoostrap(key, id string) error {
	endpoint := fmt.Sprintf("%s/%s", configsEndpoint, id)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, key, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Boostrap(key, id string) (BoostrapConfig, error) {
	endpoint := fmt.Sprintf("%s/%s", bootstrapEndpoint, id)
	url := createURL(sdk.bootstrapURL, sdk.bootstrapPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return BoostrapConfig{}, err
	}

	resp, err := sdk.sendRequest(req, key, string(CTJSON))
	if err != nil {
		return BoostrapConfig{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BoostrapConfig{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return BoostrapConfig{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var bc BoostrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BoostrapConfig{}, err
	}

	return bc, nil
}
