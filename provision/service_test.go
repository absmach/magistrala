// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
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
			sdkerr:  errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, 401),
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
		desc         string
		config       provision.Config
		domainID     string
		token        string
		clientID     string
		ttl          string
		serial       string
		cert         string
		key          string
		sdkClientErr error
		sdkCertErr   error
		sdkTokenErr  error
		err          error
	}{
		{
			desc:         "valid",
			config:       validConfig,
			domainID:     testsutil.GenerateUUID(t),
			token:        validToken,
			clientID:     testsutil.GenerateUUID(t),
			ttl:          "1h",
			cert:         "cert",
			key:          "key",
			sdkClientErr: nil,
			sdkCertErr:   nil,
			sdkTokenErr:  nil,
			err:          nil,
		},
		{
			desc: "empty token with config API key",
			config: provision.Config{
				Server: provision.ServiceConf{MgAPIKey: "key"},
				Cert:   provision.Cert{TTL: "1h"},
			},
			domainID:     testsutil.GenerateUUID(t),
			token:        "",
			clientID:     testsutil.GenerateUUID(t),
			ttl:          "1h",
			cert:         "cert",
			key:          "key",
			sdkClientErr: nil,
			sdkCertErr:   nil,
			sdkTokenErr:  nil,
			err:          nil,
		},
		{
			desc: "empty token with username and password",
			config: provision.Config{
				Server: provision.ServiceConf{
					MgUsername: "testUsername",
					MgPass:     "12345678",
					MgDomainID: testsutil.GenerateUUID(t),
				},
				Cert: provision.Cert{TTL: "1h"},
			},
			domainID:     testsutil.GenerateUUID(t),
			token:        "",
			clientID:     testsutil.GenerateUUID(t),
			ttl:          "1h",
			cert:         "cert",
			key:          "key",
			sdkClientErr: nil,
			sdkCertErr:   nil,
			sdkTokenErr:  nil,
			err:          nil,
		},
		{
			desc: "empty token with username and invalid password",
			config: provision.Config{
				Server: provision.ServiceConf{
					MgUsername: "testUsername",
					MgPass:     "12345678",
					MgDomainID: testsutil.GenerateUUID(t),
				},
				Cert: provision.Cert{TTL: "1h"},
			},
			domainID:     testsutil.GenerateUUID(t),
			token:        "",
			clientID:     testsutil.GenerateUUID(t),
			ttl:          "1h",
			cert:         "",
			key:          "",
			sdkClientErr: nil,
			sdkCertErr:   nil,
			sdkTokenErr:  errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, 401),
			err:          provision.ErrFailedToCreateToken,
		},
		{
			desc: "empty token with empty username and password",
			config: provision.Config{
				Server: provision.ServiceConf{},
				Cert:   provision.Cert{TTL: "1h"},
			},
			domainID:     testsutil.GenerateUUID(t),
			token:        "",
			clientID:     testsutil.GenerateUUID(t),
			ttl:          "1h",
			cert:         "",
			key:          "",
			sdkClientErr: nil,
			sdkCertErr:   nil,
			sdkTokenErr:  nil,
			err:          provision.ErrMissingCredentials,
		},
		{
			desc:         "invalid clientID",
			config:       validConfig,
			domainID:     testsutil.GenerateUUID(t),
			token:        "invalid",
			clientID:     testsutil.GenerateUUID(t),
			ttl:          "1h",
			cert:         "",
			key:          "",
			sdkClientErr: errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, 401),
			sdkCertErr:   nil,
			sdkTokenErr:  nil,
			err:          provision.ErrUnauthorized,
		},
		{
			desc:         "invalid clientID",
			config:       validConfig,
			domainID:     testsutil.GenerateUUID(t),
			token:        validToken,
			clientID:     "invalid",
			ttl:          "1h",
			cert:         "",
			key:          "",
			sdkClientErr: errors.NewSDKErrorWithStatus(repoerr.ErrNotFound, 404),
			sdkCertErr:   nil,
			sdkTokenErr:  nil,
			err:          provision.ErrUnauthorized,
		},
		{
			desc:         "failed to issue cert",
			config:       validConfig,
			domainID:     testsutil.GenerateUUID(t),
			token:        validToken,
			clientID:     testsutil.GenerateUUID(t),
			ttl:          "1h",
			cert:         "",
			key:          "",
			sdkClientErr: nil,
			sdkTokenErr:  nil,
			sdkCertErr:   errors.NewSDKError(repoerr.ErrCreateEntity),
			err:          repoerr.ErrCreateEntity,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			mgsdk := new(sdkmocks.SDK)
			svc := provision.New(c.config, mgsdk, mglog.NewMock())

			mgsdk.On("Client", c.clientID, c.domainID, mock.Anything).Return(sdk.Client{ID: c.clientID}, c.sdkClientErr)
			mgsdk.On("IssueCert", c.clientID, c.config.Cert.TTL, c.domainID, mock.Anything).Return(sdk.Cert{SerialNumber: c.serial}, c.sdkCertErr)
			mgsdk.On("ViewCert", c.serial, mock.Anything, mock.Anything).Return(sdk.Cert{Certificate: c.cert, Key: c.key}, c.sdkCertErr)
			login := sdk.Login{
				Identity: c.config.Server.MgUsername,
				Secret:   c.config.Server.MgPass,
			}
			mgsdk.On("CreateToken", login).Return(sdk.Token{AccessToken: validToken}, c.sdkTokenErr)
			cert, key, err := svc.Cert(c.domainID, c.token, c.clientID, c.ttl)
			assert.Equal(t, c.cert, cert)
			assert.Equal(t, c.key, key)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected error %v, got %v", c.err, err))
		})
	}
}
