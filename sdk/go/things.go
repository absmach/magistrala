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

const thingsEndpoint = "things"

func (sdk mfSDK) CreateThing(thing Thing, token string) (string, error) {
	data, err := json.Marshal(thing)
	if err != nil {
		return "", ErrInvalidArgs
	}

	url := createURL(sdk.baseURL, sdk.thingsPrefix, thingsEndpoint)

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
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return "", ErrInvalidArgs
		case http.StatusForbidden:
			return "", ErrUnauthorized
		default:
			return "", ErrFailedCreation
		}
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", thingsEndpoint))
	return id, nil
}

func (sdk mfSDK) Things(token string, offset, limit uint64, name string) (ThingsPage, error) {
	endpoint := fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", thingsEndpoint, offset, limit, name)
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ThingsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return ThingsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ThingsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ThingsPage{}, ErrInvalidArgs
		case http.StatusForbidden:
			return ThingsPage{}, ErrUnauthorized
		default:
			return ThingsPage{}, ErrFetchFailed
		}
	}

	var p thingsPageRes
	if err := json.Unmarshal(body, &p); err != nil {
		return ThingsPage{}, err
	}

	return ThingsPage{
		Things: p.Things,
		Total:  p.Total,
		Offset: p.Offset,
		Limit:  p.Limit,
	}, nil
}

func (sdk mfSDK) ThingsByChannel(token, chanID string, offset, limit uint64) (ThingsPage, error) {
	endpoint := fmt.Sprintf("channels/%s/things?offset=%d&limit=%d", chanID, offset, limit)
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ThingsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return ThingsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ThingsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ThingsPage{}, ErrInvalidArgs
		case http.StatusForbidden:
			return ThingsPage{}, ErrUnauthorized
		default:
			return ThingsPage{}, ErrFetchFailed
		}
	}

	var p thingsPageRes
	if err := json.Unmarshal(body, &p); err != nil {
		return ThingsPage{}, err
	}

	return ThingsPage{
		Things: p.Things,
		Total:  p.Total,
		Offset: p.Offset,
		Limit:  p.Limit,
	}, nil
}

func (sdk mfSDK) Thing(id, token string) (Thing, error) {
	endpoint := fmt.Sprintf("%s/%s", thingsEndpoint, id)
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Thing{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Thing{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Thing{}, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusForbidden:
			return Thing{}, ErrUnauthorized
		case http.StatusNotFound:
			return Thing{}, ErrNotFound
		default:
			return Thing{}, ErrFetchFailed
		}
	}

	var t Thing
	if err := json.Unmarshal(body, &t); err != nil {
		return Thing{}, err
	}

	return t, nil
}

func (sdk mfSDK) UpdateThing(thing Thing, token string) error {
	data, err := json.Marshal(thing)
	if err != nil {
		return ErrInvalidArgs
	}

	endpoint := fmt.Sprintf("%s/%s", thingsEndpoint, thing.ID)
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

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

func (sdk mfSDK) DeleteThing(id, token string) error {
	endpoint := fmt.Sprintf("%s/%s", thingsEndpoint, id)
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

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
		case http.StatusForbidden:
			return ErrUnauthorized
		case http.StatusBadRequest:
			return ErrInvalidArgs
		default:
			return ErrFailedRemoval
		}
	}

	return nil
}

func (sdk mfSDK) ConnectThing(thingID, chanID, token string) error {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", channelsEndpoint, chanID, thingsEndpoint, thingID)
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusForbidden:
			return ErrUnauthorized
		case http.StatusNotFound:
			return ErrNotFound
		default:
			return ErrFailedConnection
		}
	}

	return nil
}

func (sdk mfSDK) DisconnectThing(thingID, chanID, token string) error {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", channelsEndpoint, chanID, thingsEndpoint, thingID)
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

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
		case http.StatusForbidden:
			return ErrUnauthorized
		case http.StatusNotFound:
			return ErrNotFound
		default:
			return ErrFailedDisconnect
		}
	}

	return nil
}
