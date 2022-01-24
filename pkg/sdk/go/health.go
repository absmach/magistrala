// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
)

func (sdk mfSDK) Health() (mainflux.HealthInfo, error) {
	url := fmt.Sprintf("%s/health", sdk.thingsURL)

	resp, err := sdk.client.Get(url)
	if err != nil {
		return mainflux.HealthInfo{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return mainflux.HealthInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return mainflux.HealthInfo{}, errors.Wrap(ErrFetchHealth, errors.New(resp.Status))
	}

	var h mainflux.HealthInfo
	if err := json.Unmarshal(body, &h); err != nil {
		return mainflux.HealthInfo{}, err
	}

	return h, nil
}
