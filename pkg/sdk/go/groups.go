// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
)

const (
	groupsEndpoint = "groups"
	MaxLevel       = uint64(5)
	MinLevel       = uint64(1)
)

// Group represents the group of Clients.
// Indicates a level in tree hierarchy. Root node is level 1.
// Path in a tree consisting of group IDs
// Paths are unique per owner.
type Group struct {
	ID          string    `json:"id,omitempty"`
	DomainID    string    `json:"domain_id,omitempty"`
	ParentID    string    `json:"parent_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Metadata    Metadata  `json:"metadata,omitempty"`
	Level       int       `json:"level,omitempty"`
	Path        string    `json:"path,omitempty"`
	Children    []*Group  `json:"children,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Status      string    `json:"status,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
}

func (sdk mgSDK) CreateGroup(g Group, token string) (Group, errors.SDKError) {
	data, err := json.Marshal(g)
	if err != nil {
		return Group{}, errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s", sdk.usersURL, groupsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return Group{}, sdkerr
	}

	g = Group{}
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return g, nil
}

func (sdk mgSDK) Groups(pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, groupsEndpoint, pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return sdk.getGroups(url, token)
}

func (sdk mgSDK) Parents(id string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	pm.Level = MaxLevel
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/%s", sdk.usersURL, groupsEndpoint, id), "parents", pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return sdk.getGroups(url, token)
}

func (sdk mgSDK) Children(id string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	pm.Level = MaxLevel
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/%s", sdk.usersURL, groupsEndpoint, id), "children", pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return sdk.getGroups(url, token)
}

func (sdk mgSDK) getGroups(url, token string) (GroupsPage, errors.SDKError) {
	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return GroupsPage{}, err
	}

	var tp GroupsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return tp, nil
}

func (sdk mgSDK) Group(id, token string) (Group, errors.SDKError) {
	if id == "" {
		return Group{}, errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, groupsEndpoint, id)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Group{}, err
	}

	var t Group
	if err := json.Unmarshal(body, &t); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) GroupPermissions(id, token string) (Group, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.usersURL, groupsEndpoint, id, permissionsEndpoint)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Group{}, err
	}

	var t Group
	if err := json.Unmarshal(body, &t); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateGroup(g Group, token string) (Group, errors.SDKError) {
	data, err := json.Marshal(g)
	if err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	if g.ID == "" {
		return Group{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, groupsEndpoint, g.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Group{}, sdkerr
	}

	g = Group{}
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return g, nil
}

func (sdk mgSDK) EnableGroup(id, token string) (Group, errors.SDKError) {
	return sdk.changeGroupStatus(id, enableEndpoint, token)
}

func (sdk mgSDK) DisableGroup(id, token string) (Group, errors.SDKError) {
	return sdk.changeGroupStatus(id, disableEndpoint, token)
}

func (sdk mgSDK) AddUserToGroup(groupID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.usersURL, groupsEndpoint, groupID, usersEndpoint, assignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mgSDK) RemoveUserFromGroup(groupID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.usersURL, groupsEndpoint, groupID, usersEndpoint, unassignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) ListChannelUserGroups(pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, groupsEndpoint, pm)
	if err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return UsersPage{}, sdkerr
	}
	up := UsersPage{}
	if err := json.Unmarshal(body, &up); err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	return up, nil
}

func (sdk mgSDK) ListGroupChannels(groupID string, pm PageMetadata, token string) (ChannelsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.thingsURL, fmt.Sprintf("%s/%s/%s", groupsEndpoint, groupID, channelsEndpoint), pm)
	if err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ChannelsPage{}, sdkerr
	}
	cp := ChannelsPage{}
	if err := json.Unmarshal(body, &cp); err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) ListGroupChannels(groupID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.thingsURL, fmt.Sprintf("%s/%s/%s", groupsEndpoint, groupID, channelsEndpoint), pm)
	if err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ChannelsPage{}, sdkerr
	}
	cp := ChannelsPage{}
	if err := json.Unmarshal(body, &cp); err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) DeleteGroup(id, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, groupsEndpoint, id)
	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) changeGroupStatus(id, status, token string) (Group, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.usersURL, groupsEndpoint, id, status)

	_, body, err := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Group{}, err
	}
	g := Group{}
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, errors.NewSDKError(err)
	}

	return g, nil
}
