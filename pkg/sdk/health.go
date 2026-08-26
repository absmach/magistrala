// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"

	smqerrors "github.com/absmach/magistrala/pkg/errors"
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

func (sdk mgSDK) Health(service string) (HealthInfo, smqerrors.SDKError) {
	var url string
	switch service {
	case "certs":
		url = fmt.Sprintf("%s/health", sdk.certsURL)
	case "fluxmq":
		url = fmt.Sprintf("%s/health", sdk.httpAdapterURL)
	default:
		return HealthInfo{}, smqerrors.NewSDKError(fmt.Errorf("unsupported health service %q", service))
	}

	resp, err := sdk.client.Get(url)
	if err != nil {
		return HealthInfo{}, smqerrors.NewSDKError(err)
	}
	defer resp.Body.Close()

	if err := smqerrors.CheckError(resp, http.StatusOK); err != nil {
		return HealthInfo{}, err
	}

	var h HealthInfo
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return HealthInfo{}, smqerrors.NewSDKError(err)
	}

	return h, nil
}
