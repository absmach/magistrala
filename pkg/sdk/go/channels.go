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

const channelsEndpoint = "channels"

// Channel represents magistrala channel.
type Channel struct {
	ID          string     `json:"id,omitempty"`
	DomainID    string     `json:"domain_id,omitempty"`
	ParentID    string     `json:"parent_id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Metadata    Metadata   `json:"metadata,omitempty"`
	Level       int        `json:"level,omitempty"`
	Path        string     `json:"path,omitempty"`
	Children    []*Channel `json:"children,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty"`
	Status      string     `json:"status,omitempty"`
	Permissions []string   `json:"permissions,omitempty"`
}

func (sdk mgSDK) CreateChannel(c Channel, domainID, token string) (Channel, errors.SDKError) {
	data, err := json.Marshal(c)
	if err != nil {
		return Channel{}, errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return Channel{}, sdkerr
	}

	c = Channel{}
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}

func (sdk mgSDK) Channels(pm PageMetadata, domainID, token string) (ChannelsPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, channelsEndpoint)
	url, err := sdk.withQueryParams(sdk.clientsURL, endpoint, pm)
	if err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ChannelsPage{}, sdkerr
	}

	var cp ChannelsPage
	if err = json.Unmarshal(body, &cp); err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) ChannelsByClient(clientID string, pm PageMetadata, domainID, token string) (ChannelsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/clients/%s", sdk.clientsURL, domainID, clientID), channelsEndpoint, pm)
	if err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ChannelsPage{}, sdkerr
	}

	var cp ChannelsPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) Channel(id, domainID, token string) (Channel, errors.SDKError) {
	if id == "" {
		return Channel{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, id)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Channel{}, err
	}

	var c Channel
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}

func (sdk mgSDK) ChannelPermissions(id, domainID, token string) (Channel, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, id, permissionsEndpoint)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Channel{}, err
	}

	var c Channel
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}

func (sdk mgSDK) UpdateChannel(c Channel, domainID, token string) (Channel, errors.SDKError) {
	if c.ID == "" {
		return Channel{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, c.ID)

	data, err := json.Marshal(c)
	if err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Channel{}, sdkerr
	}

	c = Channel{}
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}

func (sdk mgSDK) AddUserToChannel(channelID string, req UsersRelationRequest, domainID, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, channelID, usersEndpoint, assignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mgSDK) RemoveUserFromChannel(channelID string, req UsersRelationRequest, domainID, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, channelID, usersEndpoint, unassignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) ListChannelUsers(channelID string, pm PageMetadata, domainID, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s/%s/%s", domainID, channelsEndpoint, channelID, usersEndpoint), pm)
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

func (sdk mgSDK) AddUserGroupToChannel(channelID string, req UserGroupsRequest, domainID, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, channelID, groupsEndpoint, assignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mgSDK) RemoveUserGroupFromChannel(channelID string, req UserGroupsRequest, domainID, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, channelID, groupsEndpoint, unassignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) ListChannelUserGroups(channelID string, pm PageMetadata, domainID, token string) (GroupsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s/%s/%s", domainID, channelsEndpoint, channelID, groupsEndpoint), pm)
	if err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return GroupsPage{}, sdkerr
	}
	gp := GroupsPage{}
	if err := json.Unmarshal(body, &gp); err != nil {
		return GroupsPage{}, errors.NewSDKError(err)
	}

	return gp, nil
}

func (sdk mgSDK) Connect(conn Connection, domainID, token string) errors.SDKError {
	data, err := json.Marshal(conn)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.clientsURL, domainID, connectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)

	return sdkerr
}

func (sdk mgSDK) Disconnect(connIDs Connection, domainID, token string) errors.SDKError {
	data, err := json.Marshal(connIDs)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.clientsURL, domainID, disconnectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) ConnectClient(clientID, channelID, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, channelID, clientsEndpoint, clientID, connectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusCreated)

	return sdkerr
}

func (sdk mgSDK) DisconnectClient(clientID, channelID, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, channelID, clientsEndpoint, clientID, disconnectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) EnableChannel(id, domainID, token string) (Channel, errors.SDKError) {
	return sdk.changeChannelStatus(id, enableEndpoint, domainID, token)
}

func (sdk mgSDK) DisableChannel(id, domainID, token string) (Channel, errors.SDKError) {
	return sdk.changeChannelStatus(id, disableEndpoint, domainID, token)
}

func (sdk mgSDK) DeleteChannel(id, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, id)
	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) changeChannelStatus(id, status, domainID, token string) (Channel, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, channelsEndpoint, id, status)

	_, body, err := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Channel{}, err
	}
	c := Channel{}
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}
