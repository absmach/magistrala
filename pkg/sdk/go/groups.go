// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
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

func (sdk mfSDK) CreateGroup(g Group, token string) (string, errors.SDKError) {
	data, err := json.Marshal(g)
	if err != nil {
		return "", errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s", sdk.authURL, groupsEndpoint)

	headers, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	id := strings.TrimPrefix(headers.Get("Location"), fmt.Sprintf("/%s/", groupsEndpoint))
	return id, nil
}

func (sdk mfSDK) DeleteGroup(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, id)
	_, _, err := sdk.processRequest(http.MethodDelete, url, token, string(CTJSON), nil, http.StatusNoContent)
	return err
}

func (sdk mfSDK) Assign(memberIDs []string, memberType, groupID string, token string) errors.SDKError {
	var ids []string
	url := fmt.Sprintf("%s/%s/%s/members", sdk.authURL, groupsEndpoint, groupID)
	ids = append(ids, memberIDs...)
	assignReq := assignRequest{
		Type:    memberType,
		Members: ids,
	}

	data, err := json.Marshal(assignReq)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusOK)
	return sdkerr
}

func (sdk mfSDK) Unassign(token, groupID string, memberIDs ...string) errors.SDKError {
	var ids []string
	url := fmt.Sprintf("%s/%s/%s/members", sdk.authURL, groupsEndpoint, groupID)
	ids = append(ids, memberIDs...)
	assignReq := assignRequest{
		Members: ids,
	}

	data, err := json.Marshal(assignReq)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, string(CTJSON), data, http.StatusNoContent)
	return sdkerr
}

func (sdk mfSDK) Members(groupID, token string, offset, limit uint64) (MembersPage, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/members?offset=%d&limit=%d&", sdk.authURL, groupsEndpoint, groupID, offset, limit)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return MembersPage{}, err
	}

	var tp MembersPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return MembersPage{}, errors.NewSDKError(err)
	}

	return tp, nil
}

func (sdk mfSDK) Groups(meta PageMetadata, token string) (GroupsPage, errors.SDKError) {
	u, err := url.Parse(sdk.authURL)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
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

func (sdk mfSDK) Parents(id string, offset, limit uint64, token string) (GroupsPage, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/parents?offset=%d&limit=%d&tree=false&level=%d", sdk.authURL, groupsEndpoint, id, offset, limit, MaxLevel)
	return sdk.getGroups(token, url)
}

func (sdk mfSDK) Children(id string, offset, limit uint64, token string) (GroupsPage, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/children?offset=%d&limit=%d&tree=false&level=%d", sdk.authURL, groupsEndpoint, id, offset, limit, MaxLevel)
	return sdk.getGroups(token, url)
}

func (sdk mfSDK) getGroups(token, url string) (GroupsPage, errors.SDKError) {
	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return GroupsPage{}, err
	}

	var tp GroupsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}
	return tp, nil
}

func (sdk mfSDK) Group(id, token string) (Group, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, id)
	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return Group{}, err
	}

	var t Group
	if err := json.Unmarshal(body, &t); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mfSDK) UpdateGroup(t Group, token string) errors.SDKError {
	data, err := json.Marshal(t)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, t.ID)
	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, string(CTJSON), data, http.StatusOK)

	return sdkerr
}

func (sdk mfSDK) Memberships(memberID, token string, offset, limit uint64) (GroupsPage, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/groups?offset=%d&limit=%d&", sdk.authURL, membersEndpoint, memberID, offset, limit)
	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return GroupsPage{}, err
	}

	var tp GroupsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return tp, nil
}
