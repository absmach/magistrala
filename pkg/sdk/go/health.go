// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux/pkg/errors"
)

// HealthInfo contains version endpoint response.
type HealthInfo struct {
	// Status contains service status.
	Status string `json:"status"`

	// Version contains current service version.
	Version string `json:"version"`

	// Commit represents the git hash commit.
	Commit string `json:"commit"`

	// Description contains service description.
	Description string `json:"description"`

	// BuildTime contains service build time.
	BuildTime string `json:"build_time"`
}

func (sdk mfSDK) Health() (HealthInfo, errors.SDKError) {
	url := fmt.Sprintf("%s/health", sdk.thingsURL)

	resp, err := sdk.client.Get(url)
	if err != nil {
		return HealthInfo{}, errors.NewSDKError(err)
	}
	defer resp.Body.Close()

	if err := errors.CheckError(resp, http.StatusOK); err != nil {
		return HealthInfo{}, err
	}

	var h HealthInfo
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return HealthInfo{}, errors.NewSDKError(err)
	}

	return h, nil
}
