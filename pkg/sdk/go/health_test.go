// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things/clients"
	"github.com/mainflux/mainflux/things/clients/mocks"
	gmocks "github.com/mainflux/mainflux/things/groups/mocks"
	"github.com/mainflux/mainflux/things/policies"
	pmocks "github.com/mainflux/mainflux/things/policies/mocks"
	cmocks "github.com/mainflux/mainflux/users/clients/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	thingsDescription = "things service"
	thingsStatus      = "pass"
)

func TestHealth(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	pRepo := new(pmocks.Repository)
	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	ts := newThingsServer(svc, psvc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mfsdk := sdk.NewSDK(sdkConf)
	cases := map[string]struct {
		empty bool
		err   errors.SDKError
	}{
		"get things service health check": {
			empty: false,
			err:   nil,
		},
	}
	for desc, tc := range cases {
		h, err := mfsdk.Health()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
		assert.Equal(t, thingsStatus, h.Status, fmt.Sprintf("%s: expected %s status, got %s", desc, thingsStatus, h.Status))
		assert.Equal(t, tc.empty, h.Version == "", fmt.Sprintf("%s: expected non-empty version", desc))
		assert.Equal(t, mainflux.Commit, h.Commit, fmt.Sprintf("%s: expected non-empty commit", desc))
		assert.Equal(t, thingsDescription, h.Description, fmt.Sprintf("%s: expected proper description, got %s", desc, h.Description))
		assert.Equal(t, mainflux.BuildTime, h.BuildTime, fmt.Sprintf("%s: expected default epoch date, got %s", desc, h.BuildTime))
	}
}
