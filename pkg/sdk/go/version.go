// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

type version struct {
	Value string `json:"version"`
}

func (sdk mfSDK) Version() (string, error) {
	url := fmt.Sprintf("%s/version", sdk.baseURL)

	resp, err := sdk.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Wrap(ErrFetchVersion, errors.New(resp.Status))
	}

	var ver version
	if err := json.Unmarshal(body, &ver); err != nil {
		return "", err
	}

	return ver.Value, nil
}
