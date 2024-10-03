// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policysvc "github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	validToken      = "validToken"
	invalidToken    = "invalid"
	invalidDomainID = "invalid"
	email           = "test@example.com"
	unknown         = "unknown"
	channelsNum     = 3
	instanceID      = "5de9b29a-feb9-11ed-be56-0242ac120002"
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	encKey   = []byte("1234567891011121")
	domainID = testsutil.GenerateUUID(&testing.T{})
	channel  = bootstrap.Channel{
		ID:       testsutil.GenerateUUID(&testing.T{}),
		Name:     "name",
		Metadata: map[string]interface{}{"name": "value"},
	}

	config = bootstrap.Config{
		ThingID:     testsutil.GenerateUUID(&testing.T{}),
		ThingKey:    testsutil.GenerateUUID(&testing.T{}),
		ExternalID:  testsutil.GenerateUUID(&testing.T{}),
		ExternalKey: testsutil.GenerateUUID(&testing.T{}),
		Channels:    []bootstrap.Channel{channel},
		Content:     "config",
	}
)

func enc(in []byte) ([]byte, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(in))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], in)
	return ciphertext, nil
}

func TestAdd(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	neID := config
	neID.ThingID = "non-existent"

	wrongChannels := config
	ch := channel
	ch.ID = "invalid"
	wrongChannels.Channels = append(wrongChannels.Channels, ch)

	cases := []struct {
		desc            string
		config          bootstrap.Config
		token           string
		userID          string
		domainID        string
		authResponse    *magistrala.AuthorizeRes
		authorizeErr    error
		identifyErr     error
		thingErr        error
		createThingErr  error
		channelErr      error
		listExistingErr error
		saveErr         error
		deleteThingErr  error
		err             error
	}{
		{
			desc:         "add a new config",
			config:       config,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:         "add a config with an invalid ID",
			config:       neID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			thingErr:     errors.NewSDKError(svcerr.ErrNotFound),
			err:          svcerr.ErrNotFound,
		},
		{
			desc:     "add a config with invalid token",
			config:   config,
			token:    invalidToken,
			domainID: domainID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "add a config with empty token",
			config:   config,
			token:    "",
			domainID: domainID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:            "add a config with invalid list of channels",
			config:          wrongChannels,
			token:           validToken,
			userID:          validID,
			domainID:        domainID,
			authResponse:    &magistrala.AuthorizeRes{Authorized: true},
			listExistingErr: svcerr.ErrMalformedEntity,
			err:             svcerr.ErrMalformedEntity,
		},
		{
			desc:         "add empty config",
			config:       bootstrap.Config{},
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
		},
		{
			desc:         "add a config without authorization",
			config:       config,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:        "add a config with empty domain ID",
			config:      config,
			token:       validToken,
			userID:      validID,
			domainID:    "",
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "add a config with invalid domain ID",
			config:      config,
			token:       validToken,
			userID:      validID,
			domainID:    invalidDomainID,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := sdk.On("Thing", tc.config.ThingID, tc.token).Return(mgsdk.Thing{ID: tc.config.ThingID, Credentials: mgsdk.Credentials{Secret: tc.config.ThingKey}}, tc.thingErr)
		repoCall1 := sdk.On("CreateThing", mock.Anything, tc.token).Return(mgsdk.Thing{}, tc.createThingErr)
		repoCall2 := sdk.On("DeleteThing", tc.config.ThingID, tc.token).Return(tc.deleteThingErr)
		repoCall3 := boot.On("ListExisting", context.Background(), tc.domainID, mock.Anything).Return(tc.config.Channels, tc.listExistingErr)
		repoCall4 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)

		_, err := svc.Add(context.Background(), tc.token, tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestView(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	cases := []struct {
		desc         string
		configID     string
		userID       string
		domain       string
		thingDomain  string
		authorizeRes *magistrala.AuthorizeRes
		token        string
		identifyErr  error
		authorizeErr error
		retrieveErr  error
		thingErr     error
		channelErr   error
		err          error
	}{
		{
			desc:         "view an existing config",
			configID:     config.ThingID,
			userID:       validID,
			thingDomain:  domainID,
			domain:       domainID,
			token:        validToken,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:         "view a non-existing config",
			configID:     unknown,
			userID:       validID,
			thingDomain:  domainID,
			domain:       domainID,
			token:        validToken,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			retrieveErr:  svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
		},
		{
			desc:        "view a config with wrong credentials",
			configID:    config.ThingID,
			thingDomain: domainID,
			domain:      domainID,
			token:       invalidToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "view a config with invalid domain",
			configID:    config.ThingID,
			userID:      validID,
			thingDomain: domainID,
			domain:      invalidDomainID,
			token:       validToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "view a config with invalid thing domain",
			configID:    config.ThingID,
			userID:      validID,
			thingDomain: invalidDomainID,
			domain:      domainID,
			token:       validToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:         "view a config with failed authorization",
			configID:     config.ThingID,
			userID:       validID,
			thingDomain:  domainID,
			domain:       domainID,
			token:        validToken,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domain}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeRes, tc.authorizeErr)
		repoCall := boot.On("RetrieveByID", context.Background(), tc.thingDomain, tc.configID).Return(config, tc.retrieveErr)

		_, err := svc.View(context.Background(), tc.token, tc.configID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
	}
}

func TestUpdate(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	c := config
	ch := channel
	ch.ID = "2"
	c.Channels = append(c.Channels, ch)

	modifiedCreated := c
	modifiedCreated.Content = "new-config"
	modifiedCreated.Name = "new name"

	nonExisting := c
	nonExisting.ThingID = unknown

	cases := []struct {
		desc         string
		config       bootstrap.Config
		token        string
		userID       string
		domainID     string
		authorizeRes *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		updateErr    error
		err          error
	}{
		{
			desc:         "update a config with state Created",
			config:       modifiedCreated,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:         "update a non-existing config",
			config:       nonExisting,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:    svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
		},
		{
			desc:        "update a config with wrong credentials",
			config:      c,
			token:       invalidToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:         "update a config with failed authorization",
			config:       c,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "update a config with update error",
			config:       c,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:    svcerr.ErrUpdateEntity,
			err:          svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeRes, tc.authorizeErr)
		repoCall := boot.On("Update", context.Background(), mock.Anything).Return(tc.updateErr)
		err := svc.Update(context.Background(), tc.token, tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
	}
}

func TestUpdateCert(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	c := config
	ch := channel
	ch.ID = "2"
	c.Channels = append(c.Channels, ch)

	cases := []struct {
		desc           string
		token          string
		userID         string
		domainID       string
		thingID        string
		clientCert     string
		clientKey      string
		caCert         string
		expectedConfig bootstrap.Config
		authorizeRes   *magistrala.AuthorizeRes
		authorizeErr   error
		identifyErr    error
		updateErr      error
		err            error
	}{
		{
			desc:         "update certs for the valid config",
			userID:       validID,
			domainID:     domainID,
			thingID:      c.ThingID,
			clientCert:   "newCert",
			clientKey:    "newKey",
			caCert:       "newCert",
			token:        validToken,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			expectedConfig: bootstrap.Config{
				Name:        c.Name,
				ThingKey:    c.ThingKey,
				Channels:    c.Channels,
				ExternalID:  c.ExternalID,
				ExternalKey: c.ExternalKey,
				Content:     c.Content,
				State:       c.State,
				DomainID:    c.DomainID,
				ThingID:     c.ThingID,
				ClientCert:  "newCert",
				CACert:      "newCert",
				ClientKey:   "newKey",
			},
			err: nil,
		},
		{
			desc:           "update cert for a non-existing config",
			userID:         validID,
			domainID:       domainID,
			thingID:        "empty",
			clientCert:     "newCert",
			clientKey:      "newKey",
			caCert:         "newCert",
			token:          validToken,
			authorizeRes:   &magistrala.AuthorizeRes{Authorized: true},
			expectedConfig: bootstrap.Config{},
			updateErr:      svcerr.ErrNotFound,
			err:            svcerr.ErrNotFound,
		},
		{
			desc:           "update config cert with wrong credentials",
			thingID:        c.ThingID,
			clientCert:     "newCert",
			clientKey:      "newKey",
			caCert:         "newCert",
			token:          invalidToken,
			expectedConfig: bootstrap.Config{},
			identifyErr:    svcerr.ErrAuthentication,
			err:            svcerr.ErrAuthentication,
		},
		{
			desc:           "update config cert with failed authorization",
			userID:         validID,
			domainID:       domainID,
			thingID:        c.ThingID,
			clientCert:     "newCert",
			clientKey:      "newKey",
			caCert:         "newCert",
			token:          validToken,
			authorizeRes:   &magistrala.AuthorizeRes{Authorized: false},
			expectedConfig: bootstrap.Config{},
			authorizeErr:   svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeRes, tc.authorizeErr)
		repoCall := boot.On("UpdateCert", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.expectedConfig, tc.updateErr)

		cfg, err := svc.UpdateCert(context.Background(), tc.token, tc.thingID, tc.clientCert, tc.clientKey, tc.caCert)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		sort.Slice(cfg.Channels, func(i, j int) bool {
			return cfg.Channels[i].ID < cfg.Channels[j].ID
		})
		sort.Slice(tc.expectedConfig.Channels, func(i, j int) bool {
			return tc.expectedConfig.Channels[i].ID < tc.expectedConfig.Channels[j].ID
		})
		assert.Equal(t, tc.expectedConfig, cfg, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.expectedConfig, cfg))
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
	}
}

func TestUpdateConnections(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	c := config
	c.State = bootstrap.Inactive

	activeConf := config
	activeConf.State = bootstrap.Active

	ch := channel

	nonExisting := config
	nonExisting.ThingID = unknown

	cases := []struct {
		desc         string
		token        string
		id           string
		state        bootstrap.State
		userID       string
		domainID     string
		connections  []string
		authorizeRes *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		updateErr    error
		thingErr     error
		channelErr   error
		retrieveErr  error
		listErr      error
		err          error
	}{
		{
			desc:         "update connections for config with state Inactive",
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			id:           c.ThingID,
			state:        c.State,
			connections:  []string{ch.ID},
			err:          nil,
		},
		{
			desc:         "update connections for config with state Active",
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			id:           activeConf.ThingID,
			state:        activeConf.State,
			connections:  []string{ch.ID},
			err:          nil,
		},
		{
			desc:         "update connections for non-existing config",
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: false},
			id:           nonExisting.ThingID,
			connections:  []string{"3"},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "update connections with invalid channels",
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			id:           c.ThingID,
			connections:  []string{"wrong"},
			channelErr:   errors.NewSDKError(svcerr.ErrNotFound),
			err:          svcerr.ErrNotFound,
		},
		{
			desc:        "update connections a config with wrong credentials",
			token:       invalidToken,
			id:          c.ThingID,
			connections: []string{ch.ID, "3"},
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:         "update connections a config with failed authorization",
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			id:           c.ThingID,
			connections:  []string{ch.ID, "3"},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeRes, tc.authorizeErr)
		sdkCall := sdk.On("Channel", mock.Anything, tc.token).Return(mgsdk.Channel{}, tc.channelErr)
		repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(c, tc.retrieveErr)
		repoCall1 := boot.On("ListExisting", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(c.Channels, tc.listErr)
		repoCall2 := boot.On("UpdateConnections", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.updateErr)
		err := svc.UpdateConnections(context.Background(), tc.token, tc.id, tc.connections)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		authCall1.Unset()
		sdkCall.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestList(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	numThings := 101
	var saved []bootstrap.Config
	for i := 0; i < numThings; i++ {
		c := config
		c.ExternalID = testsutil.GenerateUUID(t)
		c.ExternalKey = testsutil.GenerateUUID(t)
		c.Name = fmt.Sprintf("%s-%d", config.Name, i)
		if i == 41 {
			c.State = bootstrap.Active
		}
		saved = append(saved, c)
	}
	cases := []struct {
		desc                string
		config              bootstrap.ConfigsPage
		filter              bootstrap.Filter
		offset              uint64
		limit               uint64
		token               string
		userID              string
		domainID            string
		superAdminAuthRes   *magistrala.AuthorizeRes
		domainAdminAuthRes  *magistrala.AuthorizeRes
		superAdmiAuthErr    error
		domainAdmiAuthErr   error
		listObjectsResponse policysvc.PolicyPage
		authorizeErr        error
		identifyErr         error
		listObjectsErr      error
		retrieveErr         error
		err                 error
	}{
		{
			desc: "list configs successfully as super admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse: policysvc.PolicyPage{},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc:                "list configs with failed super admin check",
			config:              bootstrap.ConfigsPage{},
			filter:              bootstrap.Filter{},
			token:               validID,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			listObjectsResponse: policysvc.PolicyPage{},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs successfully as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: true},
			listObjectsResponse: policysvc.PolicyPage{},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc:                "list configs wit failed domain admin check",
			config:              bootstrap.ConfigsPage{},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			listObjectsResponse: policysvc.PolicyPage{},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs successfully as non admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs with specified name as super admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: saved[95:96],
			},
			filter:            bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:             validToken,
			userID:            validID,
			domainID:          domainID,
			superAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			offset:            0,
			limit:             100,
			err:               nil,
		},
		{
			desc: "list configs with specified name as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: saved[95:96],
			},
			filter:             bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			offset:             0,
			limit:              100,
			err:                nil,
		},
		{
			desc: "list configs with specified name as non admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: saved[95:96],
			},
			filter:              bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              0,
			limit:               100,
			err:                 nil,
		},
		{
			desc:        "list configs with invalid token",
			config:      bootstrap.ConfigsPage{},
			filter:      bootstrap.Filter{},
			token:       invalidToken,
			offset:      0,
			limit:       10,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "List configs with empty domain",
			config: bootstrap.ConfigsPage{
				Total:   0,
				Offset:  0,
				Limit:   10,
				Configs: []bootstrap.Config{},
			},
			filter:   bootstrap.Filter{},
			token:    validToken,
			userID:   validID,
			domainID: "",
			offset:   0,
			limit:    10,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "list last page as super admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  95,
				Limit:   10,
				Configs: saved[95:],
			},
			filter:             bootstrap.Filter{},
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: true},
			domainAdminAuthRes: &magistrala.AuthorizeRes{Authorized: false},
			offset:             95,
			limit:              10,
			err:                nil,
		},
		{
			desc: "list last page as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  95,
				Limit:   10,
				Configs: saved[95:],
			},
			filter:             bootstrap.Filter{},
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			offset:             95,
			limit:              10,
			err:                nil,
		},
		{
			desc: "list last page as non admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  95,
				Limit:   10,
				Configs: saved[95:],
			},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              95,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs with Active state as super admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:             bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: true},
			domainAdminAuthRes: &magistrala.AuthorizeRes{Authorized: false},
			offset:             35,
			limit:              20,
			err:                nil,
		},
		{
			desc: "list configs with Active state as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:             bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			offset:             35,
			limit:              20,
			err:                nil,
		},
		{
			desc: "list configs with Active state as non admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:              bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              35,
			limit:               20,
			err:                 nil,
		},
		{
			desc:                "list configs with failed to list objects",
			config:              bootstrap.ConfigsPage{},
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.userID,
			Permission:  policysvc.AdminPermission,
			ObjectType:  policysvc.PlatformType,
			Object:      policysvc.MagistralaObject,
		}).Return(tc.superAdminAuthRes, tc.superAdmiAuthErr)
		authCall2 := auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			SubjectType: policysvc.UserType,
			SubjectKind: policysvc.UsersKind,
			Subject:     tc.userID,
			Permission:  policysvc.AdminPermission,
			ObjectType:  policysvc.DomainType,
			Object:      tc.domainID,
		}).Return(tc.domainAdminAuthRes, tc.domainAdmiAuthErr)
		authCall3 := policies.On("ListAllObjects", mock.Anything, policysvc.PolicyReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.userID,
			Permission:  policysvc.ViewPermission,
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		repoCall := boot.On("RetrieveAll", context.Background(), mock.Anything, mock.Anything, tc.filter, tc.offset, tc.limit).Return(tc.config, tc.retrieveErr)

		result, err := svc.List(context.Background(), tc.token, tc.filter, tc.offset, tc.limit)
		assert.ElementsMatch(t, tc.config.Configs, result.Configs, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Configs, result.Configs))
		assert.Equal(t, tc.config.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Total, result.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		authCall1.Unset()
		authCall2.Unset()
		authCall3.Unset()
		repoCall.Unset()
	}
}

func TestRemove(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	c := config
	cases := []struct {
		desc         string
		id           string
		token        string
		userID       string
		domainID     string
		authorizeRes *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		removeErr    error
		err          error
	}{
		{
			desc:        "remove a config with wrong credentials",
			id:          c.ThingID,
			token:       invalidToken,
			domainID:    invalidDomainID,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:         "remove an existing config",
			id:           c.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:         "remove removed config",
			id:           c.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:         "remove non-existing config",
			id:           unknown,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "remove a config with failed authorization",
			id:           c.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "remove a config with failed remove",
			id:           c.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			removeErr:    svcerr.ErrRemoveEntity,
			err:          svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authorizeRes, tc.authorizeErr)
		repoCall := boot.On("Remove", context.Background(), mock.Anything, mock.Anything).Return(tc.removeErr)
		err := svc.Remove(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		authCall1.Unset()
		repoCall.Unset()
	}
}

func TestBootstrap(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	c := config
	e, err := enc([]byte(c.ExternalKey))
	assert.Nil(t, err, fmt.Sprintf("Encrypting external key expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		config      bootstrap.Config
		externalKey string
		externalID  string
		userID      string
		domainID    string
		err         error
		encrypted   bool
	}{
		{
			desc:        "bootstrap using invalid external id",
			config:      bootstrap.Config{},
			externalID:  "invalid",
			externalKey: c.ExternalKey,
			userID:      validID,
			domainID:    invalidDomainID,
			err:         svcerr.ErrNotFound,
			encrypted:   false,
		},
		{
			desc:        "bootstrap using invalid external key",
			config:      bootstrap.Config{},
			externalID:  c.ExternalID,
			externalKey: "invalid",
			userID:      validID,
			domainID:    domainID,
			err:         bootstrap.ErrExternalKey,
			encrypted:   false,
		},
		{
			desc:        "bootstrap an existing config",
			config:      c,
			externalID:  c.ExternalID,
			externalKey: c.ExternalKey,
			userID:      validID,
			domainID:    domainID,
			err:         nil,
			encrypted:   false,
		},
		{
			desc:        "bootstrap encrypted",
			config:      c,
			externalID:  c.ExternalID,
			externalKey: hex.EncodeToString(e),
			userID:      validID,
			domainID:    domainID,
			err:         nil,
			encrypted:   true,
		},
	}

	for _, tc := range cases {
		repoCall := boot.On("RetrieveByExternalID", context.Background(), mock.Anything).Return(tc.config, tc.err)
		config, err := svc.Bootstrap(context.Background(), tc.externalKey, tc.externalID, tc.encrypted)
		assert.Equal(t, tc.config, config, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.config, config))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestChangeState(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	c := config
	cases := []struct {
		desc          string
		state         bootstrap.State
		id            string
		token         string
		userID        string
		domainID      string
		identifyErr   error
		retrieveErr   error
		connectErr    errors.SDKError
		disconenctErr error
		stateErr      error
		err           error
	}{
		{
			desc:        "change state with wrong credentials",
			state:       bootstrap.Active,
			id:          c.ThingID,
			token:       invalidToken,
			domainID:    invalidDomainID,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "change state of non-existing config",
			state:       bootstrap.Active,
			id:          unknown,
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:     "change state to Active",
			state:    bootstrap.Active,
			id:       c.ThingID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "change state to current state",
			state:    bootstrap.Active,
			id:       c.ThingID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "change state to Inactive",
			state:    bootstrap.Inactive,
			id:       c.ThingID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:       "change state with failed Connect",
			state:      bootstrap.Active,
			id:         c.ThingID,
			token:      validToken,
			userID:     validID,
			domainID:   domainID,
			connectErr: errors.NewSDKError(bootstrap.ErrThings),
			err:        bootstrap.ErrThings,
		},
		{
			desc:     "change state with invalid state",
			state:    bootstrap.State(2),
			id:       c.ThingID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			stateErr: svcerr.ErrMalformedEntity,
			err:      svcerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(c, tc.retrieveErr)
		sdkCall := sdk.On("Connect", mock.Anything, mock.Anything).Return(tc.connectErr)
		repoCall1 := boot.On("ChangeState", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.stateErr)

		err := svc.ChangeState(context.Background(), tc.token, tc.id, tc.state)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		sdkCall.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	ch := bootstrap.Channel{
		ID:       channel.ID,
		Name:     "new name",
		Metadata: map[string]interface{}{"meta": "new"},
	}

	cases := []struct {
		desc    string
		channel bootstrap.Channel
		err     error
	}{
		{
			desc:    "update an existing channel",
			channel: ch,
			err:     nil,
		},
		{
			desc:    "update a non-existing channel",
			channel: bootstrap.Channel{ID: ""},
			err:     nil,
		},
	}

	for _, tc := range cases {
		repoCall := boot.On("UpdateChannel", context.Background(), mock.Anything).Return(tc.err)
		err := svc.UpdateChannelHandler(context.Background(), tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRemoveChannelHandler(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing channel",
			id:   config.Channels[0].ID,
			err:  nil,
		},
		{
			desc: "remove a non-existing channel",
			id:   "unknown",
			err:  nil,
		},
	}

	for _, tc := range cases {
		repoCall := boot.On("RemoveChannel", context.Background(), mock.Anything).Return(tc.err)
		err := svc.RemoveChannelHandler(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRemoveConfigHandler(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing config",
			id:   config.ThingID,
			err:  nil,
		},
		{
			desc: "remove a non-existing channel",
			id:   "unknown",
			err:  nil,
		},
	}

	for _, tc := range cases {
		repoCall := boot.On("RemoveThing", context.Background(), mock.Anything).Return(tc.err)
		err := svc.RemoveConfigHandler(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestConnectThingsHandler(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	cases := []struct {
		desc      string
		thingID   string
		channelID string
		err       error
	}{
		{
			desc:      "connect",
			channelID: channel.ID,
			thingID:   config.ThingID,
			err:       nil,
		},
		{
			desc:      "connect connected",
			channelID: channel.ID,
			thingID:   config.ThingID,
			err:       svcerr.ErrAddPolicies,
		},
	}

	for _, tc := range cases {
		repoCall := boot.On("ConnectThing", context.Background(), mock.Anything, mock.Anything).Return(tc.err)
		err := svc.ConnectThingHandler(context.Background(), tc.channelID, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestDisconnectThingsHandler(t *testing.T) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	policies := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policies, boot, sdk, encKey, idp)

	cases := []struct {
		desc      string
		thingID   string
		channelID string
		err       error
	}{
		{
			desc:      "disconnect",
			channelID: channel.ID,
			thingID:   config.ThingID,
			err:       nil,
		},
		{
			desc:      "disconnect disconnected",
			channelID: channel.ID,
			thingID:   config.ThingID,
			err:       nil,
		},
	}

	for _, tc := range cases {
		repoCall := boot.On("DisconnectThing", context.Background(), mock.Anything, mock.Anything).Return(tc.err)
		err := svc.DisconnectThingHandler(context.Background(), tc.channelID, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}
