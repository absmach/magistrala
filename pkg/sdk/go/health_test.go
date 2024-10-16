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
)

func TestHealth(t *testing.T) {
	thingsTs, _, _ := setupThings()
	defer thingsTs.Close()

	usersTs, _, _ := setupUsers()
	defer usersTs.Close()

	certsTs, _ := setupCerts()
	defer certsTs.Close()

	bootstrapTs, _, _ := setupBootstrap()
	defer bootstrapTs.Close()

	readerTs, _, _ := setupReader()
	defer readerTs.Close()

	httpAdapterTs, _, _ := setupMessages()
	defer httpAdapterTs.Close()

	sdkConf := sdk.Config{
		ThingsURL:       thingsTs.URL,
		UsersURL:        usersTs.URL,
		CertsURL:        certsTs.URL,
		BootstrapURL:    bootstrapTs.URL,
		ReaderURL:       readerTs.URL,
		HTTPAdapterURL:  httpAdapterTs.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)
	cases := []struct {
		desc        string
		service     string
		empty       bool
		description string
		status      string
		err         errors.SDKError
	}{
		{
			desc:        "get things service health check",
			service:     "things",
			empty:       false,
			err:         nil,
			description: "things service",
			status:      "pass",
		},
		{
			desc:        "get users service health check",
			service:     "users",
			empty:       false,
			err:         nil,
			description: "users service",
			status:      "pass",
		},
		{
			desc:        "get certs service health check",
			service:     "certs",
			empty:       false,
			err:         nil,
			description: "certs service",
			status:      "pass",
		},
		{
			desc:        "get bootstrap service health check",
			service:     "bootstrap",
			empty:       false,
			err:         nil,
			description: "bootstrap service",
			status:      "pass",
		},
		{
			desc:        "get reader service health check",
			service:     "reader",
			empty:       false,
			err:         nil,
			description: "test service",
			status:      "pass",
		},
		{
			desc:        "get http-adapter service health check",
			service:     "http-adapter",
			empty:       false,
			err:         nil,
			description: "http service",
			status:      "pass",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			h, err := mgsdk.Health(tc.service)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, h.Status, fmt.Sprintf("%s: expected %s status, got %s", tc.desc, tc.status, h.Status))
			assert.Equal(t, tc.empty, h.Version == "", fmt.Sprintf("%s: expected non-empty version", tc.desc))
			assert.Equal(t, magistrala.Commit, h.Commit, fmt.Sprintf("%s: expected non-empty commit", tc.desc))
			assert.Equal(t, tc.description, h.Description, fmt.Sprintf("%s: expected proper description, got %s", tc.desc, h.Description))
			assert.Equal(t, magistrala.BuildTime, h.BuildTime, fmt.Sprintf("%s: expected default epoch date, got %s", tc.desc, h.BuildTime))
		})
	}
}
