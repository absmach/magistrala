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

	"github.com/mainflux/mainflux"
)

// Version - server health check
func (sdk *MfxSDK) Version() (mainflux.VersionInfo, error) {
	url := fmt.Sprintf("%s/version", sdk.url)

	resp, err := sdk.httpClient.Get(url)
	if err != nil {
		return mainflux.VersionInfo{}, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return mainflux.VersionInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return mainflux.VersionInfo{}, fmt.Errorf("%d", resp.StatusCode)
	}

	var ver mainflux.VersionInfo
	if err := json.Unmarshal(body, &ver); err != nil {
		return mainflux.VersionInfo{}, err
	}
	return ver, nil
}
