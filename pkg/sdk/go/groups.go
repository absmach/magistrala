// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	groupsEndpoint = "groups"
	MaxLevel       = uint64(5)
	MinLevel       = uint64(1)
)

func (sdk mfSDK) CreateGroup(g Group, token string) (string, error) {
	data, err := json.Marshal(g)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s", sdk.authURL, groupsEndpoint)
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
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, id)
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
	url := fmt.Sprintf("%s/%s/%s/members", sdk.authURL, groupsEndpoint, groupID)
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
	url := fmt.Sprintf("%s/%s/%s/members", sdk.authURL, groupsEndpoint, groupID)
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

func (sdk mfSDK) Members(groupID, token string, offset, limit uint64) (MembersPage, error) {
	url := fmt.Sprintf("%s/%s/%s/members?offset=%d&limit=%d&", sdk.authURL, groupsEndpoint, groupID, offset, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return MembersPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return MembersPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return MembersPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return MembersPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp MembersPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return MembersPage{}, err
	}

	return tp, nil
}

func (sdk mfSDK) Groups(meta PageMetadata, token string) (GroupsPage, error) {
	u, err := url.Parse(sdk.authURL)
	if err != nil {
		return GroupsPage{}, err
	}
	u.Path = groupsEndpoint
	q := u.Query()
	q.Add("offset", strconv.FormatUint(meta.Offset, 10))
	if meta.Limit != 0 {
		q.Add("limit", strconv.FormatUint(meta.Limit, 10))
	}
	if meta.Level != 0 {
		q.Add("level", strconv.FormatUint(meta.Level, 10))
	}
	if meta.Name != "" {
		q.Add("name", meta.Name)
	}
	if meta.Type != "" {
		q.Add("type", meta.Type)
	}
	u.RawQuery = q.Encode()
	return sdk.getGroups(token, u.String())
}

func (sdk mfSDK) Parents(id string, offset, limit uint64, token string) (GroupsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/parents?offset=%d&limit=%d&tree=false&level=%d", sdk.authURL, groupsEndpoint, id, offset, limit, MaxLevel)
	return sdk.getGroups(token, url)
}

func (sdk mfSDK) Children(id string, offset, limit uint64, token string) (GroupsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/children?offset=%d&limit=%d&tree=false&level=%d", sdk.authURL, groupsEndpoint, id, offset, limit, MaxLevel)
	return sdk.getGroups(token, url)
}

func (sdk mfSDK) getGroups(token, url string) (GroupsPage, error) {
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
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, id)
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

	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, t.ID)
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

func (sdk mfSDK) Memberships(memberID, token string, offset, limit uint64) (GroupsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/groups?offset=%d&limit=%d&", sdk.authURL, membersEndpoint, memberID, offset, limit)
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
