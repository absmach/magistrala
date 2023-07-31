// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	configsEndpoint        = "configs"
	bootstrapEndpoint      = "bootstrap"
	whitelistEndpoint      = "state"
	bootstrapCertsEndpoint = "configs/certs"
	bootstrapConnEndpoint  = "configs/connections"
	secureEndpoint         = "secure"
)

// BootstrapConfig represents Configuration entity. It wraps information about external entity
// as well as info about corresponding Mainflux entities.
// MFThing represents corresponding Mainflux Thing ID.
// MFKey is key of corresponding Mainflux Thing.
// MFChannels is a list of Mainflux Channels corresponding Mainflux Thing connects to.
type BootstrapConfig struct {
	Channels    interface{} `json:"channels,omitempty"`
	ExternalID  string      `json:"external_id,omitempty"`
	ExternalKey string      `json:"external_key,omitempty"`
	ThingID     string      `json:"thing_id,omitempty"`
	ThingKey    string      `json:"thing_key,omitempty"`
	Name        string      `json:"name,omitempty"`
	ClientCert  string      `json:"client_cert,omitempty"`
	ClientKey   string      `json:"client_key,omitempty"`
	CACert      string      `json:"ca_cert,omitempty"`
	Content     string      `json:"content,omitempty"`
	State       int         `json:"state,omitempty"`
}

func (ts *BootstrapConfig) UnmarshalJSON(data []byte) error {
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return err
	}

	if channelData, ok := rawData["channels"]; ok {
		var stringData []string
		if err := json.Unmarshal(channelData, &stringData); err == nil {
			ts.Channels = stringData
		} else {
			var channels []Channel
			if err := json.Unmarshal(channelData, &channels); err == nil {
				ts.Channels = channels
			} else {
				return fmt.Errorf("unsupported channel data type")
			}
		}
	}

	if err := json.Unmarshal(data, &struct {
		ExternalID  *string `json:"external_id,omitempty"`
		ExternalKey *string `json:"external_key,omitempty"`
		ThingID     *string `json:"thing_id,omitempty"`
		ThingKey    *string `json:"thing_key,omitempty"`
		Name        *string `json:"name,omitempty"`
		ClientCert  *string `json:"client_cert,omitempty"`
		ClientKey   *string `json:"client_key,omitempty"`
		CACert      *string `json:"ca_cert,omitempty"`
		Content     *string `json:"content,omitempty"`
		State       *int    `json:"state,omitempty"`
	}{
		ExternalID:  &ts.ExternalID,
		ExternalKey: &ts.ExternalKey,
		ThingID:     &ts.ThingID,
		ThingKey:    &ts.ThingKey,
		Name:        &ts.Name,
		ClientCert:  &ts.ClientCert,
		ClientKey:   &ts.ClientKey,
		CACert:      &ts.CACert,
		Content:     &ts.Content,
		State:       &ts.State,
	}); err != nil {
		return err
	}

	return nil
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

func (sdk mfSDK) Bootstraps(pm PageMetadata, token string) (BootstrapPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.bootstrapURL, configsEndpoint, pm)
	if err != nil {
		return BootstrapPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if sdkerr != nil {
		return BootstrapPage{}, sdkerr
	}

	var bb BootstrapPage
	if err = json.Unmarshal(body, &bb); err != nil {
		return BootstrapPage{}, errors.NewSDKError(err)
	}

	return bb, nil
}

func (sdk mfSDK) Whitelist(cfg BootstrapConfig, token string) errors.SDKError {
	data, err := json.Marshal(BootstrapConfig{State: cfg.State})
	if err != nil {
		return errors.NewSDKError(err)
	}

	if cfg.ThingID == "" {
		return errors.NewSDKError(errors.ErrNotFoundParam)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, whitelistEndpoint, cfg.ThingID)

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

	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, configsEndpoint, cfg.ThingID)
	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, string(CTJSON), data, http.StatusOK)

	return sdkerr
}

func (sdk mfSDK) UpdateBootstrapCerts(id, clientCert, clientKey, ca, token string) (BootstrapConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, bootstrapCertsEndpoint, id)
	request := BootstrapConfig{
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     ca,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, string(CTJSON), data, http.StatusOK)

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, sdkerr
}

func (sdk mfSDK) UpdateBootstrapConnection(id string, channels []string, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, bootstrapConnEndpoint, id)
	request := map[string][]string{
		"channels": channels,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, string(CTJSON), data, http.StatusOK)
	return sdkerr
}

func (sdk mfSDK) RemoveBootstrap(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, configsEndpoint, id)
	_, _, err := sdk.processRequest(http.MethodDelete, url, token, string(CTJSON), nil, http.StatusNoContent)
	return err
}

func (sdk mfSDK) Bootstrap(externalID, externalKey string) (BootstrapConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, bootstrapEndpoint, externalID)
	_, body, err := sdk.processRequest(http.MethodGet, url, ThingPrefix+externalKey, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mfSDK) BootstrapSecure(externalID, externalKey string) (BootstrapConfig, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, bootstrapEndpoint, secureEndpoint, externalID)
	_, body, err := sdk.processRequest(http.MethodGet, url, ThingPrefix+externalKey, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}
