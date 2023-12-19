// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
)

const channelsEndpoint = "channels"

// Channel represents magistrala channel.
type Channel struct {
	ID          string     `json:"id,omitempty"`
	OwnerID     string     `json:"owner_id,omitempty"`
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

func (sdk mgSDK) CreateChannel(c Channel, token string) (Channel, errors.SDKError) {
	data, err := json.Marshal(c)
	if err != nil {
		return Channel{}, errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s", sdk.thingsURL, channelsEndpoint)

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

func (sdk mgSDK) CreateChannels(chs []Channel, token string) ([]Channel, errors.SDKError) {
	data, err := json.Marshal(chs)
	if err != nil {
		return []Channel{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, "bulk")

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return []Channel{}, sdkerr
	}

	var ccr createChannelsRes
	if err := json.Unmarshal(body, &ccr); err != nil {
		return []Channel{}, errors.NewSDKError(err)
	}

	return ccr.Channels, nil
}

func (sdk mgSDK) Channels(pm PageMetadata, token string) (ChannelsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.thingsURL, channelsEndpoint, pm)
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

func (sdk mgSDK) ChannelsByThing(thingID string, pm PageMetadata, token string) (ChannelsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/things/%s", sdk.thingsURL, thingID), channelsEndpoint, pm)
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

func (sdk mgSDK) Channel(id, token string) (Channel, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, id)

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

func (sdk mgSDK) ChannelPermissions(id, token string) (Channel, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, id, permissionsEndpoint)

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

func (sdk mgSDK) UpdateChannel(c Channel, token string) (Channel, errors.SDKError) {
	data, err := json.Marshal(c)
	if err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, c.ID)

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

func (sdk mgSDK) AddUserToChannel(channelID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID, usersEndpoint, assignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mgSDK) RemoveUserFromChannel(channelID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID, usersEndpoint, unassignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) ListChannelUsers(channelID string, pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s/%s", channelsEndpoint, channelID, usersEndpoint), pm)
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

func (sdk mgSDK) AddUserGroupToChannel(channelID string, req UserGroupsRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID, groupsEndpoint, assignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mgSDK) RemoveUserGroupFromChannel(channelID string, req UserGroupsRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID, groupsEndpoint, unassignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) ListChannelUserGroups(channelID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s/%s", channelsEndpoint, channelID, groupsEndpoint), pm)
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

func (sdk mgSDK) Connect(conn Connection, token string) errors.SDKError {
	data, err := json.Marshal(conn)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, connectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)

	return sdkerr
}

func (sdk mgSDK) Disconnect(connIDs Connection, token string) errors.SDKError {
	data, err := json.Marshal(connIDs)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, disconnectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) ConnectThing(thingID, channelID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID, thingsEndpoint, thingID, connectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusCreated)

	return sdkerr
}

func (sdk mgSDK) DisconnectThing(thingID, channelID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID, thingsEndpoint, thingID, disconnectEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) EnableChannel(id, token string) (Channel, errors.SDKError) {
	return sdk.changeChannelStatus(id, enableEndpoint, token)
}

func (sdk mgSDK) DisableChannel(id, token string) (Channel, errors.SDKError) {
	return sdk.changeChannelStatus(id, disableEndpoint, token)
}

func (sdk mgSDK) DeleteChannel(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, id)
	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) changeChannelStatus(id, status, token string) (Channel, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, id, status)

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
