// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
)

const thingsEndpoint = "things"
const connectEndpoint = "connect"

func (sdk mfSDK) CreateThing(t Thing, token string) (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", err
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
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", thingsEndpoint))
	return id, nil
}

func (sdk mfSDK) CreateThings(things []Thing, token string) ([]Thing, error) {
	data, err := json.Marshal(things)
	if err != nil {
		return []Thing{}, err
	}

	endpoint := fmt.Sprintf("%s/%s", thingsEndpoint, "bulk")
	url := createURL(sdk.baseURL, sdk.thingsPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return []Thing{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return []Thing{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return []Thing{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Thing{}, err
	}

	var ctr createThingsRes
	if err := json.Unmarshal(body, &ctr); err != nil {
		return []Thing{}, err
	}

	return ctr.Things, nil
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
		return ThingsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp ThingsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return ThingsPage{}, err
	}

	return tp, nil
}

func (sdk mfSDK) ThingsByChannel(token, chanID string, offset, limit uint64, disconn bool) (ThingsPage, error) {
	endpoint := fmt.Sprintf("channels/%s/things?offset=%d&limit=%d&disconnected=%t", chanID, offset, limit, disconn)
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
		return ThingsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp ThingsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return ThingsPage{}, err
	}

	return tp, nil
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
		return Thing{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var t Thing
	if err := json.Unmarshal(body, &t); err != nil {
		return Thing{}, err
	}

	return t, nil
}

func (sdk mfSDK) UpdateThing(t Thing, token string) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s", thingsEndpoint, t.ID)
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
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
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
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Connect(connIDs ConnectionIDs, token string) error {
	data, err := json.Marshal(connIDs)
	if err != nil {
		return err
	}

	url := createURL(sdk.baseURL, sdk.thingsPrefix, connectEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedConnect, errors.New(resp.Status))
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
		return errors.Wrap(ErrFailedDisconnect, errors.New(resp.Status))
	}

	return nil
}
