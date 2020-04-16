// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/mainflux/mainflux/errors"
)

func (sdk mfSDK) CreateUser(u User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}

	url := createURL(sdk.baseURL, sdk.usersPrefix, "users")

	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
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
		return User{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
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
		return "", err
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
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	var tr tokenRes
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}

	return tr.Token, nil
}

func (sdk mfSDK) UpdateUser(u User, token string) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
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
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
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
		return err
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
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
	}

	return nil
}
