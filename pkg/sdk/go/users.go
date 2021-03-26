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

const (
	usersEndpoint    = "users"
	tokensEndpoint   = "tokens"
	passwordEndpoint = "password"
	membersEndpoint  = "members"
)

func (sdk mfSDK) CreateUser(u User) (string, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return "", err
	}

	url := createURL(sdk.baseURL, sdk.usersPrefix, usersEndpoint)

	resp, err := sdk.client.Post(url, string(CTJSON), bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", usersEndpoint))
	return id, nil
}

func (sdk mfSDK) User(token string) (User, error) {
	url := createURL(sdk.baseURL, sdk.usersPrefix, usersEndpoint)

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

	url := createURL(sdk.baseURL, sdk.usersPrefix, tokensEndpoint)

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

	url := createURL(sdk.baseURL, sdk.usersPrefix, usersEndpoint)

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

	url := createURL(sdk.baseURL, sdk.usersPrefix, passwordEndpoint)

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
func (sdk mfSDK) Memberships(memberID, token string, offset, limit uint64) (GroupsPage, error) {
	endpoint := fmt.Sprintf("%s/%s/groups?offset=%d&limit=%d&", membersEndpoint, memberID, offset, limit)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return GroupsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return GroupsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GroupsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return GroupsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp GroupsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return GroupsPage{}, err
	}

	return tp, nil
}
