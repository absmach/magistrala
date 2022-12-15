// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
)

func (sdk mfSDK) Health() (mainflux.HealthInfo, errors.SDKError) {
	url := fmt.Sprintf("%s/health", sdk.thingsURL)

	resp, err := sdk.client.Get(url)
	if err != nil {
		return mainflux.HealthInfo{}, errors.NewSDKError(err)
	}
	defer resp.Body.Close()

	if err := errors.CheckError(resp, http.StatusOK); err != nil {
		return mainflux.HealthInfo{}, err
	}

	var h mainflux.HealthInfo
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return mainflux.HealthInfo{}, errors.NewSDKError(err)
	}

	return h, nil
}
