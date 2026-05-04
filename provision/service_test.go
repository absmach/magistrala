// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	smqSDK "github.com/absmach/magistrala/pkg/sdk"
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
		content map[string]any
		sdkerr  error
		err     error
	}{
		{
			desc:    "valid request",
			content: validConfig.Bootstrap.Content,
			sdkerr:  nil,
			err:     nil,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			content := svc.Mapping()
			assert.Equal(t, c.content, content)
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
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			svc := provision.New(c.config, mgsdk, mglog.NewMock())

			call1 := mgsdk.On("Client", mock.Anything, c.clientID, c.domainID, mock.Anything).Return(smqSDK.Client{ID: c.clientID}, c.sdkClientErr)
			var call2 *mock.Call
			switch c.token {
			case "":
				call2 = mgsdk.On("IssueCert", context.Background(), c.clientID, c.config.Cert.TTL, []string{}, smqSDK.Options{}, c.domainID, c.returnedToken).Return(smqSDK.Certificate{SerialNumber: c.serial}, c.sdkCertErr)
			default:
				call2 = mgsdk.On("IssueCert", context.Background(), c.clientID, c.config.Cert.TTL, []string{}, smqSDK.Options{}, c.domainID, c.token).Return(smqSDK.Certificate{SerialNumber: c.serial}, c.sdkCertErr)
			}
			call3 := mgsdk.On("ViewCert", mock.Anything, c.serial, mock.Anything, mock.Anything).Return(smqSDK.Certificate{Certificate: c.cert, Key: c.key}, c.sdkCertErr)

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

func TestProvisionUsesBootstrapEnrollmentID(t *testing.T) {
	cfg := validConfig
	cfg.Bootstrap = provision.Bootstrap{
		X509Provision: true,
		Provision:     true,
		AutoWhiteList: true,
		Content: map[string]any{
			"broker": "mqtt://localhost:1883",
		},
	}
	cfg.Clients[0].Metadata = map[string]any{
		"external_id": "placeholder",
	}
	cfg.Channels[0].Name = "control-channel"
	cfg.Channels[0].Metadata = map[string]any{
		"type": "control",
	}
	cfg.Cert.TTL = "1h"

	const (
		name        = "gateway-1"
		externalID  = "AA:BB:CC:DD"
		externalKey = "secret"
		certPEM     = "cert-pem"
		keyPEM      = "key-pem"
		serial      = "serial-1"
	)

	clientID := testsutil.GenerateUUID(t)
	channelID := testsutil.GenerateUUID(t)
	bootstrapID := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)

	clientMetadata := map[string]any{
		"external_id": externalID,
	}
	var updatedClient smqSDK.Client

	mgsdk := new(sdkmocks.SDK)
	svc := provision.New(cfg, mgsdk, mglog.NewMock())

	createClientCall := mgsdk.On(
		"CreateClient",
		mock.Anything,
		mock.Anything,
		domainID,
		validToken,
	).Return(smqSDK.Client{ID: clientID}, nil)

	clientCall := mgsdk.On(
		"Client",
		mock.Anything,
		clientID,
		domainID,
		validToken,
	).Return(smqSDK.Client{ID: clientID, Name: name, Metadata: clientMetadata}, nil).Twice()

	createChannelCall := mgsdk.On(
		"CreateChannel",
		mock.Anything,
		mock.Anything,
		domainID,
		validToken,
	).Return(smqSDK.Channel{ID: channelID}, nil)

	channelCall := mgsdk.On(
		"Channel",
		mock.Anything,
		channelID,
		domainID,
		validToken,
	).Return(smqSDK.Channel{ID: channelID, Metadata: smqSDK.Metadata{"type": "control"}}, nil)

	addBootstrapCall := mgsdk.On(
		"AddBootstrap",
		mock.Anything,
		mock.Anything,
		domainID,
		validToken,
	).Return(bootstrapID, nil)

	viewBootstrapCall := mgsdk.On(
		"ViewBootstrap",
		mock.Anything,
		bootstrapID,
		domainID,
		validToken,
	).Return(smqSDK.BootstrapConfig{
		ID:          bootstrapID,
		ExternalID:  externalID,
		ExternalKey: externalKey,
	}, nil)

	issueCertCall := mgsdk.On(
		"IssueCert",
		mock.Anything,
		clientID,
		cfg.Cert.TTL,
		mock.Anything,
		mock.Anything,
		domainID,
		validToken,
	).Return(smqSDK.Certificate{SerialNumber: serial}, nil)

	viewCertCall := mgsdk.On(
		"ViewCert",
		mock.Anything,
		serial,
		domainID,
		validToken,
	).Return(smqSDK.Certificate{Certificate: certPEM, Key: keyPEM}, nil)

	updateBootstrapCertsCall := mgsdk.On(
		"UpdateBootstrapCerts",
		mock.Anything,
		bootstrapID,
		certPEM,
		keyPEM,
		"",
		domainID,
		validToken,
	).Return(smqSDK.BootstrapConfig{
		ID:          bootstrapID,
		ExternalID:  externalID,
		ExternalKey: externalKey,
		ClientCert:  certPEM,
		ClientKey:   keyPEM,
	}, nil)

	whitelistCall := mgsdk.On(
		"Whitelist",
		mock.Anything,
		bootstrapID,
		smqSDK.BootstrapEnabledStatus,
		domainID,
		validToken,
	).Return(nil)

	updateClientCall := mgsdk.On(
		"UpdateClient",
		mock.Anything,
		mock.Anything,
		domainID,
		validToken,
	).Run(func(args mock.Arguments) {
		updatedClient = args.Get(1).(smqSDK.Client)
	}).Return(smqSDK.Client{ID: clientID}, nil)

	res, err := svc.Provision(context.Background(), domainID, validToken, name, externalID, externalKey)
	assert.NoError(t, err)
	assert.Len(t, res.Clients, 1)
	assert.Len(t, res.Channels, 1)
	assert.True(t, res.Whitelisted[bootstrapID])
	assert.Equal(t, certPEM, res.ClientCert[clientID])
	assert.Equal(t, keyPEM, res.ClientKey[clientID])
	assert.Equal(t, clientID, updatedClient.ID)
	assert.Equal(t, bootstrapID, updatedClient.Metadata["cfg_id"])
	assert.Equal(t, externalID, updatedClient.Metadata["external_id"])
	assert.Equal(t, channelID, updatedClient.Metadata["ctrl_channel_id"])
	assert.Equal(t, "gateway", updatedClient.Metadata["type"])

	createClientCall.Unset()
	clientCall.Unset()
	createChannelCall.Unset()
	channelCall.Unset()
	addBootstrapCall.Unset()
	viewBootstrapCall.Unset()
	issueCertCall.Unset()
	viewCertCall.Unset()
	updateBootstrapCertsCall.Unset()
	whitelistCall.Unset()
	updateClientCall.Unset()
}
