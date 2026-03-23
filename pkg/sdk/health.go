// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/absmach/supermq/pkg/errors"
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

func (sdk mgSDK) Health(service string) (HealthInfo, errors.SDKError) {
	var url string
	switch service {
	case "clients":
		url = fmt.Sprintf("%s/health", sdk.clientsURL)
	case "users":
		url = fmt.Sprintf("%s/health", sdk.usersURL)
	case "certs":
		url = fmt.Sprintf("%s/health", sdk.certsURL)
	case "groups":
		url = fmt.Sprintf("%s/health", sdk.groupsURL)
	case "channels":
		url = fmt.Sprintf("%s/health", sdk.channelsURL)
	case "domains":
		url = fmt.Sprintf("%s/health", sdk.domainsURL)
	case "journal":
		url = fmt.Sprintf("%s/health", sdk.journalURL)
	case "fluxmq":
		url = fmt.Sprintf("%s/health", sdk.httpAdapterURL)
	}

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
