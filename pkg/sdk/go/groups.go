// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func (sdk mfSDK) Assign(memberIDs []string, memberType, groupID, token string) errors.SDKError {
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

func (sdk mfSDK) Unassign(groupID string, memberIDs []string, token string) errors.SDKError {
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

func (sdk mfSDK) Members(groupID string, pm PageMetadata, token string) (MembersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, groupID), "members", pm)
	if err != nil {
		return MembersPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if sdkerr != nil {
		return MembersPage{}, sdkerr
	}

	var tp MembersPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return MembersPage{}, errors.NewSDKError(err)
	}

	return tp, nil
}

func (sdk mfSDK) Groups(pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.authURL, groupsEndpoint, pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}
	return sdk.getGroups(url, token)
}

func (sdk mfSDK) Parents(id string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	pm.Level = MaxLevel
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, id), "parents", pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}
	return sdk.getGroups(url, token)
}

func (sdk mfSDK) Children(id string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	pm.Level = MaxLevel
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/%s", sdk.authURL, groupsEndpoint, id), "children", pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}
	return sdk.getGroups(url, token)
}

func (sdk mfSDK) getGroups(url, token string) (GroupsPage, errors.SDKError) {
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

func (sdk mfSDK) Memberships(memberID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/%s", sdk.authURL, membersEndpoint, memberID), groupsEndpoint, pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if sdkerr != nil {
		return GroupsPage{}, sdkerr
	}

	var tp GroupsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return tp, nil
}
