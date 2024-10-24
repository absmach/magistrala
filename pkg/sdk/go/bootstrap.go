// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
)

const (
	configsEndpoint        = "clients/configs"
	bootstrapEndpoint      = "clients/bootstrap"
	whitelistEndpoint      = "clients/state"
	bootstrapCertsEndpoint = "clients/configs/certs"
	bootstrapConnEndpoint  = "clients/configs/connections"
	secureEndpoint         = "secure"
)

// BootstrapConfig represents Configuration entity. It wraps information about external entity
// as well as info about corresponding Magistrala entities.
// MGClient represents corresponding Magistrala Client ID.
// MGKey is key of corresponding Magistrala Client.
// MGChannels is a list of Magistrala Channels corresponding Magistrala Client connects to.
type BootstrapConfig struct {
	Channels     interface{} `json:"channels,omitempty"`
	ExternalID   string      `json:"external_id,omitempty"`
	ExternalKey  string      `json:"external_key,omitempty"`
	ClientID     string      `json:"client_id,omitempty"`
	ClientSecret string      `json:"client_secret,omitempty"`
	Name         string      `json:"name,omitempty"`
	ClientCert   string      `json:"client_cert,omitempty"`
	ClientKey    string      `json:"client_key,omitempty"`
	CACert       string      `json:"ca_cert,omitempty"`
	Content      string      `json:"content,omitempty"`
	State        int         `json:"state,omitempty"`
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
		ExternalID   *string `json:"external_id,omitempty"`
		ExternalKey  *string `json:"external_key,omitempty"`
		ClientID     *string `json:"client_id,omitempty"`
		ClientSecret *string `json:"client_secret,omitempty"`
		Name         *string `json:"name,omitempty"`
		ClientCert   *string `json:"client_cert,omitempty"`
		ClientKey    *string `json:"client_key,omitempty"`
		CACert       *string `json:"ca_cert,omitempty"`
		Content      *string `json:"content,omitempty"`
		State        *int    `json:"state,omitempty"`
	}{
		ExternalID:   &ts.ExternalID,
		ExternalKey:  &ts.ExternalKey,
		ClientID:     &ts.ClientID,
		ClientSecret: &ts.ClientSecret,
		Name:         &ts.Name,
		ClientCert:   &ts.ClientCert,
		ClientKey:    &ts.ClientKey,
		CACert:       &ts.CACert,
		Content:      &ts.Content,
		State:        &ts.State,
	}); err != nil {
		return err
	}

	return nil
}

func (sdk mgSDK) AddBootstrap(cfg BootstrapConfig, domainID, token string) (string, errors.SDKError) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint)

	headers, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	id := strings.TrimPrefix(headers.Get("Location"), "/clients/configs/")

	return id, nil
}

func (sdk mgSDK) Bootstraps(pm PageMetadata, domainID, token string) (BootstrapPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, configsEndpoint)
	url, err := sdk.withQueryParams(sdk.bootstrapURL, endpoint, pm)
	if err != nil {
		return BootstrapPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return BootstrapPage{}, sdkerr
	}

	var bb BootstrapPage
	if err = json.Unmarshal(body, &bb); err != nil {
		return BootstrapPage{}, errors.NewSDKError(err)
	}

	return bb, nil
}

func (sdk mgSDK) Whitelist(clientID string, state int, domainID, token string) errors.SDKError {
	if clientID == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}

	data, err := json.Marshal(BootstrapConfig{State: state})
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, whitelistEndpoint, clientID)

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusCreated, http.StatusOK)

	return sdkerr
}

func (sdk mgSDK) ViewBootstrap(id, domainID, token string) (BootstrapConfig, errors.SDKError) {
	if id == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint, id)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mgSDK) UpdateBootstrap(cfg BootstrapConfig, domainID, token string) errors.SDKError {
	if cfg.ClientID == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint, cfg.ClientID)

	data, err := json.Marshal(cfg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusOK)

	return sdkerr
}

func (sdk mgSDK) UpdateBootstrapCerts(id, clientCert, clientKey, ca, domainID, token string) (BootstrapConfig, errors.SDKError) {
	if id == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, bootstrapCertsEndpoint, id)
	request := BootstrapConfig{
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     ca,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return BootstrapConfig{}, sdkerr
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mgSDK) UpdateBootstrapConnection(id string, channels []string, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, bootstrapConnEndpoint, id)
	request := map[string][]string{
		"channels": channels,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusOK)
	return sdkerr
}

func (sdk mgSDK) RemoveBootstrap(id, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint, id)

	_, _, err := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return err
}

func (sdk mgSDK) Bootstrap(externalID, externalKey string) (BootstrapConfig, errors.SDKError) {
	if externalID == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, bootstrapEndpoint, externalID)

	_, body, err := sdk.processRequest(http.MethodGet, url, ClientPrefix+externalKey, nil, nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mgSDK) BootstrapSecure(externalID, externalKey, cryptoKey string) (BootstrapConfig, errors.SDKError) {
	if externalID == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, bootstrapEndpoint, secureEndpoint, externalID)

	encExtKey, err := bootstrapEncrypt([]byte(externalKey), cryptoKey)
	if err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(http.MethodGet, url, ClientPrefix+encExtKey, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return BootstrapConfig{}, sdkErr
	}

	decBody, decErr := bootstrapDecrypt(body, cryptoKey)
	if decErr != nil {
		return BootstrapConfig{}, errors.NewSDKError(decErr)
	}
	var bc BootstrapConfig
	if err := json.Unmarshal(decBody, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func bootstrapEncrypt(in []byte, cryptoKey string) (string, error) {
	block, err := aes.NewCipher([]byte(cryptoKey))
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, aes.BlockSize+len(in))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], in)
	return hex.EncodeToString(ciphertext), nil
}

func bootstrapDecrypt(in []byte, cryptoKey string) ([]byte, error) {
	ciphertext := in

	block, err := aes.NewCipher([]byte(cryptoKey))
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, err
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
