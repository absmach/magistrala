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

const groupsEndpoint = "groups"

func (sdk mfSDK) CreateGroup(g Group, token string) (string, error) {
	data, err := json.Marshal(g)
	if err != nil {
		return "", err
	}

	url := createURL(sdk.baseURL, sdk.groupsPrefix, groupsEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", groupsEndpoint))
	return id, nil
}

func (sdk mfSDK) DeleteGroup(id, token string) error {
	endpoint := fmt.Sprintf("%s/%s", groupsEndpoint, id)

	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Assign(userID, groupID, token string) error {
	endpoint := fmt.Sprintf("%s/%s/users/%s", groupsEndpoint, groupID, userID)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedUserAdd, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Unassign(userID, groupID, token string) error {
	endpoint := fmt.Sprintf("%s/%s/users/%s", groupsEndpoint, groupID, userID)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))
	if err != nil {
		return err
	}
	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Members(groupID, token string, offset, limit uint64) (UsersPage, error) {
	endpoint := fmt.Sprintf("%s/%s/users?offset=%d&limit=%d&", groupsEndpoint, groupID, offset, limit)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return UsersPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return UsersPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UsersPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return UsersPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp UsersPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return UsersPage{}, err
	}

	return tp, nil
}

func (sdk mfSDK) Groups(token string, offset, limit uint64, id string) (GroupsPage, error) {
	endpoint := fmt.Sprintf("%s?offset=%d&limit=%d", groupsEndpoint, offset, limit)
	if id != "" {
		endpoint = fmt.Sprintf("%s/%s/groups?offset=%d&limit=%d", groupsEndpoint, id, offset, limit)
	}
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

func (sdk mfSDK) Group(id, token string) (Group, error) {
	endpoint := fmt.Sprintf("%s/%s", groupsEndpoint, id)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Group{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Group{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Group{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Group{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var t Group
	if err := json.Unmarshal(body, &t); err != nil {
		return Group{}, err
	}

	return t, nil
}

func (sdk mfSDK) UpdateGroup(t Group, token string) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", groupsEndpoint, t.ID)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)

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
