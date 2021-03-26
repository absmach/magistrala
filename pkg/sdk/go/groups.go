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

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

const groupsEndpoint = "groups"

type assignRequest struct {
	Type    string   `json:"type,omitempty"`
	Members []string `json:"members"`
}

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

func (sdk mfSDK) Assign(memberIDs []string, memberType, groupID string, token string) error {
	var ids []string
	endpoint := fmt.Sprintf("%s/%s/members", groupsEndpoint, groupID)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)

	ids = append(ids, memberIDs...)
	assignReq := assignRequest{
		Type:    memberType,
		Members: ids,
	}

	data, err := json.Marshal(assignReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrMemberAdd, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Unassign(token, groupID string, memberIDs ...string) error {
	var ids []string
	endpoint := fmt.Sprintf("%s/%s/members", groupsEndpoint, groupID)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)

	ids = append(ids, memberIDs...)
	assignReq := assignRequest{
		Members: ids,
	}

	data, err := json.Marshal(assignReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader(data))
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

func (sdk mfSDK) Members(groupID, token string, offset, limit uint64) (auth.MemberPage, error) {
	endpoint := fmt.Sprintf("%s/%s/members?offset=%d&limit=%d&", groupsEndpoint, groupID, offset, limit)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return auth.MemberPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return auth.MemberPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return auth.MemberPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return auth.MemberPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp auth.MemberPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return auth.MemberPage{}, err
	}

	return tp, nil
}

func (sdk mfSDK) Groups(offset, limit uint64, token string) (auth.GroupPage, error) {
	endpoint := fmt.Sprintf("%s?offset=%d&limit=%d&tree=false", groupsEndpoint, offset, limit)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)
	return sdk.getGroups(token, url)
}

func (sdk mfSDK) Parents(id string, offset, limit uint64, token string) (auth.GroupPage, error) {
	endpoint := fmt.Sprintf("%s/%s/parents?offset=%d&limit=%d&tree=false&level=%d", groupsEndpoint, id, offset, limit, auth.MaxLevel)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)
	return sdk.getGroups(token, url)
}

func (sdk mfSDK) Children(id string, offset, limit uint64, token string) (auth.GroupPage, error) {
	endpoint := fmt.Sprintf("%s/%s/children?offset=%d&limit=%d&tree=false&level=%d", groupsEndpoint, id, offset, limit, auth.MaxLevel)
	url := createURL(sdk.baseURL, sdk.groupsPrefix, endpoint)
	return sdk.getGroups(token, url)
}

func (sdk mfSDK) getGroups(token, url string) (auth.GroupPage, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return auth.GroupPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return auth.GroupPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return auth.GroupPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return auth.GroupPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp auth.GroupPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return auth.GroupPage{}, err
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
