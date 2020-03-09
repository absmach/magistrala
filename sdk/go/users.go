// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func (sdk mfSDK) CreateUser(user User) error {
	if err := user.validate(); err != nil {
		return err
	}

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

func (sdk mfSDK) User(token string) (User, error) {
	url := createURL(sdk.baseURL, sdk.usersPrefix, "users")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return User{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return User{}, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusForbidden:
			return User{}, ErrUnauthorized
		default:
			return User{}, ErrFetchFailed
		}
	}

	var u User
	if err := json.Unmarshal(body, &u); err != nil {
		return User{}, err
	}

	return u, nil
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

func (sdk mfSDK) UpdateUser(user User, token string) error {
	data, err := json.Marshal(user)
	if err != nil {
		return ErrInvalidArgs
	}

	url := createURL(sdk.baseURL, sdk.usersPrefix, "users")

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusForbidden:
			return ErrUnauthorized
		default:
			return ErrFailedUpdate
		}
	}

	return nil
}

func (sdk mfSDK) UpdatePassword(oldPass, newPass, token string) error {
	ur := UserPasswordReq{
		OldPassword: oldPass,
		Password:    newPass,
	}
	data, err := json.Marshal(ur)
	if err != nil {
		return ErrInvalidArgs
	}

	url := createURL(sdk.baseURL, sdk.usersPrefix, "password")

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ErrInvalidArgs
		case http.StatusForbidden:
			return ErrUnauthorized
		default:
			return ErrFailedUpdate
		}
	}

	return nil
}
