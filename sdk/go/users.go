//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func (sdk mfSDK) CreateUser(user User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return ErrInvalidArgs
	}

	url := createURL(sdk.baseURL, sdk.usersPrefix, "users")

	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ErrInvalidArgs
		case http.StatusConflict:
			return ErrConflict
		default:
			return ErrFailedCreation
		}
	}

	return nil
}

func (sdk mfSDK) CreateToken(user User) (string, error) {
	data, err := json.Marshal(user)
	if err != nil {
		return "", ErrInvalidArgs
	}

	url := createURL(sdk.baseURL, sdk.usersPrefix, "tokens")

	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return "", ErrInvalidArgs
		case http.StatusForbidden:
			return "", ErrUnauthorized
		default:
			return "", ErrFailedCreation
		}
	}

	var t tokenRes
	if err := json.Unmarshal(body, &t); err != nil {
		return "", err
	}

	return t.Token, nil
}
