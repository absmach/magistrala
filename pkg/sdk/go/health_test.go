// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	ths, auth := setupThingsMinimal()
	auth.Test(t)
	defer ths.Close()

	usclsv, _, _, auth := setupUsers()
	auth.Test(t)
	defer usclsv.Close()

	CertTs, _, _, err := setupCerts()
	require.Nil(t, err, fmt.Sprintf("unexpected error during creating service: %s", err))
	defer CertTs.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ths.URL,
		UsersURL:        usclsv.URL,
		CertsURL:        CertTs.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)
	cases := map[string]struct {
		service     string
		empty       bool
		description string
		status      string
		err         errors.SDKError
	}{
		"get things service health check": {
			service:     "things",
			empty:       false,
			err:         nil,
			description: "things service",
			status:      "pass",
		},
		"get users service health check": {
			service:     "users",
			empty:       false,
			err:         nil,
			description: "users service",
			status:      "pass",
		},
		"get certs service health check": {
			service:     "certs",
			empty:       false,
			err:         nil,
			description: "certs service",
			status:      "pass",
		},
	}
	for desc, tc := range cases {
		h, err := mgsdk.Health(tc.service)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
		assert.Equal(t, tc.status, h.Status, fmt.Sprintf("%s: expected %s status, got %s", desc, tc.status, h.Status))
		assert.Equal(t, tc.empty, h.Version == "", fmt.Sprintf("%s: expected non-empty version", desc))
		assert.Equal(t, magistrala.Commit, h.Commit, fmt.Sprintf("%s: expected non-empty commit", desc))
		assert.Equal(t, tc.description, h.Description, fmt.Sprintf("%s: expected proper description, got %s", desc, h.Description))
		assert.Equal(t, magistrala.BuildTime, h.BuildTime, fmt.Sprintf("%s: expected default epoch date, got %s", desc, h.BuildTime))
	}
}
