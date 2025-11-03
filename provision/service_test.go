// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision_test

import (
	"context"
	"fmt"
	"testing"

	csdk "github.com/absmach/certs/sdk"
	csdkmocks "github.com/absmach/certs/sdk/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/provision"
	smqlog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	smqSDK "github.com/absmach/supermq/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var validToken = "valid"

func TestMapping(t *testing.T) {
	mgsdk := new(sdkmocks.SDK)
	certs := new(csdkmocks.SDK)
	svc := provision.New(validConfig, mgsdk, certs, smqlog.NewMock())

	cases := []struct {
		desc    string
		token   string
		content map[string]any
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
			content: map[string]any{},
			sdkerr:  errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, 401),
			err:     provision.ErrUnauthorized,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			pm := smqSDK.PageMetadata{Offset: uint64(0), Limit: uint64(10)}
			repocall := mgsdk.On("Users", mock.Anything, pm, c.token).Return(smqSDK.UsersPage{}, c.sdkerr)
			content, err := svc.Mapping(context.Background(), c.token)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected error %v, got %v", c.err, err))
			assert.Equal(t, c.content, content)
			repocall.Unset()
		})
	}
}

func TestCert(t *testing.T) {
	cases := []struct {
		desc          string
		config        provision.Config
		domainID      string
		token         string
		returnedToken string
		clientID      string
		ttl           string
		serial        string
		cert          string
		key           string
		sdkClientErr  error
		sdkCertErr    error
		sdkTokenErr   error
		err           error
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
			domainID:      testsutil.GenerateUUID(t),
			token:         "",
			returnedToken: "key",
			clientID:      testsutil.GenerateUUID(t),
			ttl:           "1h",
			cert:          "cert",
			key:           "key",
			sdkClientErr:  nil,
			sdkCertErr:    nil,
			sdkTokenErr:   nil,
			err:           nil,
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
			domainID:      testsutil.GenerateUUID(t),
			token:         "",
			returnedToken: validToken,
			clientID:      testsutil.GenerateUUID(t),
			ttl:           "1h",
			cert:          "cert",
			key:           "key",
			sdkClientErr:  nil,
			sdkCertErr:    nil,
			sdkTokenErr:   nil,
			err:           nil,
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
	mgsdk := new(sdkmocks.SDK)
	certs := new(csdkmocks.SDK)
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			svc := provision.New(c.config, mgsdk, certs, smqlog.NewMock())

			call1 := mgsdk.On("Client", mock.Anything, c.clientID, c.domainID, mock.Anything).Return(smqSDK.Client{ID: c.clientID}, c.sdkClientErr)
			var call2 *mock.Call
			switch c.token {
			case "":
				call2 = certs.On("IssueCert", context.Background(), c.clientID, c.config.Cert.TTL, []string{}, csdk.Options{}, c.domainID, c.returnedToken).Return(csdk.Certificate{SerialNumber: c.serial}, c.sdkCertErr)
			default:
				call2 = certs.On("IssueCert", context.Background(), c.clientID, c.config.Cert.TTL, []string{}, csdk.Options{}, c.domainID, c.token).Return(csdk.Certificate{SerialNumber: c.serial}, c.sdkCertErr)
			}
			call3 := certs.On("ViewCert", mock.Anything, c.serial, mock.Anything, mock.Anything).Return(csdk.Certificate{Certificate: c.cert, Key: c.key}, c.sdkCertErr)

			login := smqSDK.Login{
				Username: c.config.Server.MgUsername,
				Password: c.config.Server.MgPass,
			}
			call4 := mgsdk.On("CreateToken", mock.Anything, login).Return(smqSDK.Token{AccessToken: validToken}, c.sdkTokenErr)
			cert, key, err := svc.Cert(context.Background(), c.domainID, c.token, c.clientID, c.ttl)
			assert.Equal(t, c.cert, cert)
			assert.Equal(t, c.key, key)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected error %v, got %v", c.err, err))
			call1.Unset()
			call2.Unset()
			call3.Unset()
			call4.Unset()
		})
	}
}
