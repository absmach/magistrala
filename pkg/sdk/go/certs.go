// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

const certsEndpoint = "certs"

// Cert represents certs data.
type Cert struct {
	CACert     string `json:"issuing_ca,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
}

func (sdk mfSDK) IssueCert(thingID string, keyBits int, keyType, valid, token string) (Cert, errors.SDKError) {
	r := certReq{
		ThingID: thingID,
		KeyBits: keyBits,
		KeyType: keyType,
		Valid:   valid,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Cert{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.certsURL, certsEndpoint)
	resp, err := request(http.MethodPost, token, url, d)
	if err != nil {
		return Cert{}, errors.NewSDKError(err)
	}
	defer resp.Body.Close()

	if err := errors.CheckError(resp, http.StatusOK); err != nil {
		return Cert{}, err
	}

	var c Cert
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return Cert{}, errors.NewSDKError(err)
	}
	return c, nil
}

func (sdk mfSDK) RemoveCert(id, token string) errors.SDKError {
	resp, err := request(http.MethodDelete, token, fmt.Sprintf("%s/%s", sdk.certsURL, id), nil)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return errors.NewSDKError(err)
	}
	switch resp.StatusCode {
	case http.StatusForbidden:
		return errors.NewSDKError(errors.ErrAuthorization)
	default:
		return errors.CheckError(resp, http.StatusNoContent)
	}
}

func (sdk mfSDK) RevokeCert(thingID, certID string, token string) errors.SDKError {
	panic("not implemented")
}

func request(method, jwt, url string, data []byte) (*http.Response, errors.SDKError) {
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, errors.NewSDKError(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", jwt)
	c := &http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return nil, errors.NewSDKError(err)
	}

	return res, nil
}

type certReq struct {
	ThingID    string `json:"thing_id"`
	KeyBits    int    `json:"key_bits"`
	KeyType    string `json:"key_type"`
	Encryption string `json:"encryption"`
	Valid      string `json:"valid"`
}
