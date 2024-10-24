// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
)

const (
	certsEndpoint   = "certs"
	serialsEndpoint = "serials"
)

// Cert represents certs data.
type Cert struct {
	SerialNumber string    `json:"serial_number,omitempty"`
	Certificate  string    `json:"certificate,omitempty"`
	Key          string    `json:"key,omitempty"`
	Revoked      bool      `json:"revoked,omitempty"`
	ExpiryTime   time.Time `json:"expiry_time,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
}

func (sdk mgSDK) IssueCert(clientID, validity, domainID, token string) (Cert, errors.SDKError) {
	r := certReq{
		ClientID: clientID,
		Validity: validity,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Cert{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.certsURL, domainID, certsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, d, nil, http.StatusCreated)
	if sdkerr != nil {
		return Cert{}, sdkerr
	}

	var c Cert
	if err := json.Unmarshal(body, &c); err != nil {
		return Cert{}, errors.NewSDKError(err)
	}
	return c, nil
}

func (sdk mgSDK) ViewCert(id, domainID, token string) (Cert, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.certsURL, domainID, certsEndpoint, id)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Cert{}, err
	}

	var cert Cert
	if err := json.Unmarshal(body, &cert); err != nil {
		return Cert{}, errors.NewSDKError(err)
	}

	return cert, nil
}

func (sdk mgSDK) ViewCertByClient(clientID, domainID, token string) (CertSerials, errors.SDKError) {
	if clientID == "" {
		return CertSerials{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.certsURL, domainID, serialsEndpoint, clientID)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return CertSerials{}, err
	}
	var cs CertSerials
	if err := json.Unmarshal(body, &cs); err != nil {
		return CertSerials{}, errors.NewSDKError(err)
	}

	return cs, nil
}

func (sdk mgSDK) RevokeCert(id, domainID, token string) (time.Time, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.certsURL, domainID, certsEndpoint, id)

	_, body, err := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return time.Time{}, err
	}

	var rcr revokeCertsRes
	if err := json.Unmarshal(body, &rcr); err != nil {
		return time.Time{}, errors.NewSDKError(err)
	}

	return rcr.RevocationTime, nil
}

type certReq struct {
	ClientID string `json:"client_id"`
	Validity string `json:"ttl"`
}
