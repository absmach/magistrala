// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
)

type keyReq struct {
	Type     uint32        `json:"type,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

const keysEndpoint = "keys"

const (
	// LoginKey is temporary User key received on successfull login.
	LoginKey uint32 = iota
	// RecoveryKey represents a key for resseting password.
	RecoveryKey
	// APIKey enables the one to act on behalf of the user.
	APIKey
)

func (sdk mfSDK) Issue(token string, d time.Duration) (KeyRes, errors.SDKError) {
	datareq := keyReq{Type: APIKey, Duration: d}
	data, err := json.Marshal(datareq)
	if err != nil {
		return KeyRes{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.authURL, keysEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusCreated)
	if sdkerr != nil {
		return KeyRes{}, sdkerr
	}

	var key KeyRes
	if err := json.Unmarshal(body, &key); err != nil {
		return KeyRes{}, errors.NewSDKError(err)
	}

	return key, nil
}

func (sdk mfSDK) Revoke(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, keysEndpoint, id)
	_, _, err := sdk.processRequest(http.MethodDelete, url, token, string(CTJSON), nil, http.StatusNoContent)
	return err
}

func (sdk mfSDK) RetrieveKey(id, token string) (retrieveKeyRes, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, keysEndpoint, id)
	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return retrieveKeyRes{}, err
	}

	var key retrieveKeyRes
	if err := json.Unmarshal(body, &key); err != nil {
		return retrieveKeyRes{}, errors.NewSDKError(err)
	}

	return key, nil
}
