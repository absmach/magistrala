// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/provision"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var validToken = "valid"

func TestMapping(t *testing.T) {
	mgsdk := new(sdkmocks.SDK)
	svc := provision.New(validConfig, mgsdk, mglog.NewMock())

	cases := []struct {
		desc    string
		token   string
		content map[string]interface{}
		sdkerr  error
		err     error
	}{
		{
			desc:    "valid token",
			token:   validToken,
			content: validConfig.Bootstrap.Content,
			sdkerr:  nil,
			err:     nil,
		},
		{
			desc:    "invalid token",
			token:   "invalid",
			content: map[string]interface{}{},
			sdkerr:  errors.NewSDKErrorWithStatus(errors.ErrAuthentication, 401),
			err:     provision.ErrUnauthorized,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			pm := sdk.PageMetadata{Offset: uint64(0), Limit: uint64(10)}
			repocall := mgsdk.On("Users", pm, c.token).Return(sdk.UsersPage{}, c.sdkerr)
			content, err := svc.Mapping(c.token)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected error %v, got %v", c.err, err))
			assert.Equal(t, c.content, content)
			repocall.Unset()
		})
	}
}

func TestCert(t *testing.T) {
	cases := []struct {
		desc        string
		config      provision.Config
		token       string
		thingID     string
		ttl         string
		cert        string
		key         string
		sdkThingErr error
		sdkCertErr  error
		sdkTokenErr error
		err         error
	}{
		{
			desc:        "valid",
			config:      validConfig,
			token:       validToken,
			thingID:     testsutil.GenerateUUID(t),
			ttl:         "1h",
			cert:        "cert",
			key:         "key",
			sdkThingErr: nil,
			sdkCertErr:  nil,
			sdkTokenErr: nil,
			err:         nil,
		},
		{
			desc: "empty token with config API key",
			config: provision.Config{
				Server: provision.ServiceConf{MgAPIKey: "key"},
				Cert:   provision.Cert{TTL: "1h"},
			},
			token:       "",
			thingID:     testsutil.GenerateUUID(t),
			ttl:         "1h",
			cert:        "cert",
			key:         "key",
			sdkThingErr: nil,
			sdkCertErr:  nil,
			sdkTokenErr: nil,
			err:         nil,
		},
		{
			desc: "empty token with username and password",
			config: provision.Config{
				Server: provision.ServiceConf{
					MgUser:     "test@example.com",
					MgPass:     "12345678",
					MgDomainID: testsutil.GenerateUUID(t),
				},
				Cert: provision.Cert{TTL: "1h"},
			},
			token:       "",
			thingID:     testsutil.GenerateUUID(t),
			ttl:         "1h",
			cert:        "cert",
			key:         "key",
			sdkThingErr: nil,
			sdkCertErr:  nil,
			sdkTokenErr: nil,
			err:         nil,
		},
		{
			desc: "empty token with username and invalid password",
			config: provision.Config{
				Server: provision.ServiceConf{
					MgUser:     "test@example.com",
					MgPass:     "12345678",
					MgDomainID: testsutil.GenerateUUID(t),
				},
				Cert: provision.Cert{TTL: "1h"},
			},
			token:       "",
			thingID:     testsutil.GenerateUUID(t),
			ttl:         "1h",
			cert:        "",
			key:         "",
			sdkThingErr: nil,
			sdkCertErr:  nil,
			sdkTokenErr: errors.NewSDKErrorWithStatus(errors.ErrAuthentication, 401),
			err:         provision.ErrFailedToCreateToken,
		},
		{
			desc: "empty token with empty username and password",
			config: provision.Config{
				Server: provision.ServiceConf{},
				Cert:   provision.Cert{TTL: "1h"},
			},
			token:       "",
			thingID:     testsutil.GenerateUUID(t),
			ttl:         "1h",
			cert:        "",
			key:         "",
			sdkThingErr: nil,
			sdkCertErr:  nil,
			sdkTokenErr: nil,
			err:         provision.ErrMissingCredentials,
		},
		{
			desc:        "invalid thingID",
			config:      validConfig,
			token:       "invalid",
			thingID:     testsutil.GenerateUUID(t),
			ttl:         "1h",
			cert:        "",
			key:         "",
			sdkThingErr: errors.NewSDKErrorWithStatus(errors.ErrAuthentication, 401),
			sdkCertErr:  nil,
			sdkTokenErr: nil,
			err:         provision.ErrUnauthorized,
		},
		{
			desc:        "invalid thingID",
			config:      validConfig,
			token:       validToken,
			thingID:     "invalid",
			ttl:         "1h",
			cert:        "",
			key:         "",
			sdkThingErr: errors.NewSDKErrorWithStatus(errors.ErrNotFound, 404),
			sdkCertErr:  nil,
			sdkTokenErr: nil,
			err:         provision.ErrUnauthorized,
		},
		{
			desc:        "failed to issue cert",
			config:      validConfig,
			token:       validToken,
			thingID:     testsutil.GenerateUUID(t),
			ttl:         "1h",
			cert:        "",
			key:         "",
			sdkThingErr: nil,
			sdkTokenErr: nil,
			sdkCertErr:  errors.NewSDKError(errors.ErrCreateEntity),
			err:         errors.ErrCreateEntity,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			mgsdk := new(sdkmocks.SDK)
			svc := provision.New(c.config, mgsdk, mglog.NewMock())

			mgsdk.On("Thing", c.thingID, mock.Anything).Return(sdk.Thing{ID: c.thingID}, c.sdkThingErr)
			mgsdk.On("IssueCert", c.thingID, c.config.Cert.TTL, mock.Anything).Return(sdk.Cert{ClientCert: c.cert, ClientKey: c.key}, c.sdkCertErr)
			login := sdk.Login{
				Identity: c.config.Server.MgUser,
				Secret:   c.config.Server.MgPass,
				DomainID: c.config.Server.MgDomainID,
			}
			mgsdk.On("CreateToken", login).Return(sdk.Token{AccessToken: validToken}, c.sdkTokenErr)
			cert, key, err := svc.Cert(c.token, c.thingID, c.ttl)
			assert.Equal(t, c.cert, cert)
			assert.Equal(t, c.key, key)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected error %v, got %v", c.err, err))
		})
	}
}
