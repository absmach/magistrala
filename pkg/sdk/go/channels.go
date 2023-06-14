// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
)

const channelsEndpoint = "channels"

// Channel represents mainflux channel.
type Channel struct {
	ID          string     `json:"id"`
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
}

func (sdk mfSDK) CreateChannel(c Channel, token string) (Channel, errors.SDKError) {
	data, err := json.Marshal(c)
	if err != nil {
		return Channel{}, errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s", sdk.thingsURL, channelsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusCreated)
	if sdkerr != nil {
		return Channel{}, sdkerr
	}

	c = Channel{}
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}

func (sdk mfSDK) CreateChannels(chs []Channel, token string) ([]Channel, errors.SDKError) {
	data, err := json.Marshal(chs)
	if err != nil {
		return []Channel{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, "bulk")

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusOK)
	if sdkerr != nil {
		return []Channel{}, sdkerr
	}

	var ccr createChannelsRes
	if err := json.Unmarshal(body, &ccr); err != nil {
		return []Channel{}, errors.NewSDKError(err)
	}

	return ccr.Channels, nil
}

func (sdk mfSDK) Channels(pm PageMetadata, token string) (ChannelsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.thingsURL, channelsEndpoint, pm)
	if err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if sdkerr != nil {
		return ChannelsPage{}, sdkerr
	}

	var cp ChannelsPage
	if err = json.Unmarshal(body, &cp); err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mfSDK) ChannelsByThing(thingID string, pm PageMetadata, token string) (ChannelsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/things/%s", sdk.thingsURL, thingID), channelsEndpoint, pm)
	if err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if sdkerr != nil {
		return ChannelsPage{}, sdkerr
	}

	var cp ChannelsPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return ChannelsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mfSDK) Channel(id, token string) (Channel, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, id)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return Channel{}, err
	}

	var c Channel
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}

func (sdk mfSDK) UpdateChannel(c Channel, token string) (Channel, errors.SDKError) {
	data, err := json.Marshal(c)
	if err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, c.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPut, url, token, string(CTJSON), data, http.StatusOK)
	if sdkerr != nil {
		return Channel{}, sdkerr
	}

	c = Channel{}
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}

func (sdk mfSDK) EnableChannel(id, token string) (Channel, errors.SDKError) {
	return sdk.changeChannelStatus(id, enableEndpoint, token)
}

func (sdk mfSDK) DisableChannel(id, token string) (Channel, errors.SDKError) {
	return sdk.changeChannelStatus(id, disableEndpoint, token)
}

func (sdk mfSDK) changeChannelStatus(id, status, token string) (Channel, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, id, status)

	_, body, err := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return Channel{}, err
	}
	c := Channel{}
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, errors.NewSDKError(err)
	}

	return c, nil
}
