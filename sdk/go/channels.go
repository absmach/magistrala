//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/mainflux/mainflux/things"
)

const channelsEndpoint = "channels"

// CreateChannel - creates new channel and generates UUID
func (sdk *MfxSDK) CreateChannel(data, token string) (string, error) {
	url := fmt.Sprintf("%s/%s", sdk.url, channelsEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, token, contentTypeJSON)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("%d", resp.StatusCode)
	}

	return resp.Header.Get("Location"), nil
}

// Channels - gets all channels
func (sdk *MfxSDK) Channels(token string) ([]things.Channel, error) {
	url := fmt.Sprintf("%s/%s?offset=%s&limit=%s",
		sdk.url, channelsEndpoint, strconv.Itoa(offset), strconv.Itoa(limit))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.sendRequest(req, token, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d", resp.StatusCode)
	}

	l := listChannelsRes{}
	if err := json.Unmarshal(body, &l); err != nil {
		return nil, err
	}
	return l.Channels, nil
}

// Channel - gets channel by ID
func (sdk *MfxSDK) Channel(id, token string) (things.Channel, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.url, channelsEndpoint, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return things.Channel{}, err
	}

	resp, err := sdk.sendRequest(req, token, contentTypeJSON)
	if err != nil {
		return things.Channel{}, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return things.Channel{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return things.Channel{}, fmt.Errorf("%d", resp.StatusCode)
	}

	c := things.Channel{}
	if err := json.Unmarshal(body, &c); err != nil {
		return things.Channel{}, err
	}
	return c, nil
}

// UpdateChannel - update a channel
func (sdk *MfxSDK) UpdateChannel(id, data, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.url, channelsEndpoint, id)
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, contentTypeJSON)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%d", resp.StatusCode)
	}

	return nil
}

// DeleteChannel - removes channel
func (sdk *MfxSDK) DeleteChannel(id, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.url, channelsEndpoint, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, contentTypeJSON)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%d", resp.StatusCode)
	}

	return nil
}
