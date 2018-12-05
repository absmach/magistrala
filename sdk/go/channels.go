//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const channelsEndpoint = "channels"

func (sdk mfSDK) CreateChannel(channel Channel, token string) (string, error) {
	data, err := json.Marshal(channel)
	if err != nil {
		return "", ErrInvalidArgs
	}

	url := createURL(sdk.url, sdk.thingsPrefix, channelsEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return "", ErrInvalidArgs
		case http.StatusForbidden:
			return "", ErrUnauthorized
		default:
			return "", ErrFailedCreation
		}
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", channelsEndpoint))
	return id, nil
}

func (sdk mfSDK) Channels(token string, offset, limit uint64) ([]Channel, error) {
	endpoint := fmt.Sprintf("%s?offset=%d&limit=%d", channelsEndpoint, offset, limit)
	url := createURL(sdk.url, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return nil, ErrInvalidArgs
		case http.StatusForbidden:
			return nil, ErrUnauthorized
		default:
			return nil, ErrFetchFailed
		}
	}

	var l listChannelsRes
	if err := json.Unmarshal(body, &l); err != nil {
		return nil, err
	}

	return l.Channels, nil
}

func (sdk mfSDK) Channel(id, token string) (Channel, error) {
	endpoint := fmt.Sprintf("%s/%s", channelsEndpoint, id)
	url := createURL(sdk.url, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Channel{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Channel{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Channel{}, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusForbidden:
			return Channel{}, ErrUnauthorized
		case http.StatusNotFound:
			return Channel{}, ErrNotFound
		default:
			return Channel{}, ErrFetchFailed
		}
	}

	var c Channel
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, err
	}

	return c, nil
}

func (sdk mfSDK) UpdateChannel(channel Channel, token string) error {
	data, err := json.Marshal(channel)
	if err != nil {
		return ErrInvalidArgs
	}

	endpoint := fmt.Sprintf("%s/%s", channelsEndpoint, channel.ID)
	url := createURL(sdk.url, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ErrInvalidArgs
		case http.StatusForbidden:
			return ErrUnauthorized
		case http.StatusNotFound:
			return ErrNotFound
		default:
			return ErrFailedUpdate
		}
	}

	return nil
}

func (sdk mfSDK) DeleteChannel(id, token string) error {
	endpoint := fmt.Sprintf("%s/%s", channelsEndpoint, id)
	url := createURL(sdk.url, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ErrInvalidArgs
		case http.StatusForbidden:
			return ErrUnauthorized
		default:
			return ErrFailedUpdate
		}
	}

	return nil
}
