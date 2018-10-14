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
	"strings"
)

// CreateUser - create user
func (sdk *MfxSDK) CreateUser(user, pwd string) error {
	msg := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, user, pwd)
	url := fmt.Sprintf("%s/users", sdk.url)

	resp, err := sdk.httpClient.Post(url, contentTypeJSON, strings.NewReader(msg))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%d", resp.StatusCode)
	}

	return nil
}

// CreateToken - create user token
func (sdk *MfxSDK) CreateToken(user, pwd string) (string, error) {
	msg := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, user, pwd)
	url := fmt.Sprintf("%s/tokens", sdk.url)

	resp, err := sdk.httpClient.Post(url, contentTypeJSON, strings.NewReader(msg))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("%d", resp.StatusCode)
	}

	var t tokenRes
	if err := json.Unmarshal(body, &t); err != nil {
		return "", err
	}
	return t.Token, nil
}
