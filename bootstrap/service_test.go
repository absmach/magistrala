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
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	validToken   = "validToken"
	invalidToken = "invalid"
	email        = "test@example.com"
	unknown      = "unknown"
	channelsNum  = 3
	instanceID   = "5de9b29a-feb9-11ed-be56-0242ac120002"
	validID      = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	encKey = []byte("1234567891011121")

	channel = bootstrap.Channel{
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

func newService() (bootstrap.Service, *authmocks.Service, *sdkmocks.SDK) {
	things := mocks.NewConfigsRepository()
	auth := new(authmocks.Service)
	sdk := new(sdkmocks.SDK)

	return bootstrap.New(auth, things, sdk, encKey), auth, sdk
}

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
	svc, auth, sdk := newService()
	neID := config
	neID.ThingID = "non-existent"

	wrongChannels := config
	ch := channel
	ch.ID = "invalid"
	wrongChannels.Channels = append(wrongChannels.Channels, ch)

	cases := []struct {
		desc   string
		config bootstrap.Config
		token  string
		err    error
	}{
		{
			desc:   "add a new config",
			config: config,
			token:  validToken,
			err:    nil,
		},
		{
			desc:   "add a config with an invalid ID",
			config: neID,
			token:  validToken,
			err:    svcerr.ErrNotFound,
		},
		{
			desc:   "add a config with wrong credentials",
			config: config,
			token:  invalidToken,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "add a config with invalid list of channels",
			config: wrongChannels,
			token:  validToken,
			err:    svcerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: tc.config.ThingID, Credentials: mgsdk.Credentials{Secret: tc.config.ThingKey}}, errors.NewSDKError(tc.err))
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, errors.NewSDKError(tc.err))
		_, err := svc.Add(context.Background(), tc.token, tc.config)
		switch err {
		case nil:
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error : %s", tc.desc, err))
		default:
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestView(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "view an existing config",
			id:    saved.ThingID,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "view a non-existing config",
			id:    unknown,
			token: validToken,
			err:   svcerr.ErrNotFound,
		},
		{
			desc:  "view a config with wrong credentials",
			id:    config.ThingID,
			token: invalidToken,
			err:   svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		_, err := svc.View(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestUpdate(t *testing.T) {
	svc, auth, sdk := newService()
	c := config

	ch := channel
	ch.ID = "2"
	c.Channels = append(c.Channels, ch)

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	modifiedCreated := saved
	modifiedCreated.Content = "new-config"
	modifiedCreated.Name = "new name"

	nonExisting := config
	nonExisting.ThingID = unknown

	cases := []struct {
		desc   string
		config bootstrap.Config
		token  string
		err    error
	}{
		{
			desc:   "update a config with state Created",
			config: modifiedCreated,
			token:  validToken,
			err:    nil,
		},
		{
			desc:   "update a non-existing config",
			config: nonExisting,
			token:  validToken,
			err:    svcerr.ErrNotFound,
		},
		{
			desc:   "update a config with wrong credentials",
			config: saved,
			token:  invalidToken,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		err := svc.Update(context.Background(), tc.token, tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestUpdateCert(t *testing.T) {
	svc, auth, sdk := newService()
	c := config

	ch := channel
	ch.ID = "2"
	c.Channels = append(c.Channels, ch)
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc           string
		token          string
		thingID        string
		clientCert     string
		clientKey      string
		caCert         string
		expectedConfig bootstrap.Config
		err            error
	}{
		{
			desc:       "update certs for the valid config",
			thingID:    saved.ThingID,
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",
			token:      validToken,
			expectedConfig: bootstrap.Config{
				Name:        saved.Name,
				ThingKey:    saved.ThingKey,
				Channels:    saved.Channels,
				ExternalID:  saved.ExternalID,
				ExternalKey: saved.ExternalKey,
				Content:     saved.Content,
				State:       saved.State,
				Owner:       saved.Owner,
				ThingID:     saved.ThingID,
				ClientCert:  "newCert",
				CACert:      "newCert",
				ClientKey:   "newKey",
			},
			err: nil,
		},
		{
			desc:           "update cert for a non-existing config",
			thingID:        "empty",
			clientCert:     "newCert",
			clientKey:      "newKey",
			caCert:         "newCert",
			token:          validToken,
			expectedConfig: bootstrap.Config{},
			err:            svcerr.ErrNotFound,
		},
		{
			desc:           "update config cert with wrong credentials",
			thingID:        saved.ThingID,
			clientCert:     "newCert",
			clientKey:      "newKey",
			caCert:         "newCert",
			token:          invalidToken,
			expectedConfig: bootstrap.Config{},
			err:            svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		cfg, err := svc.UpdateCert(context.Background(), tc.token, tc.thingID, tc.clientCert, tc.clientKey, tc.caCert)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		sort.Slice(cfg.Channels, func(i, j int) bool {
			return cfg.Channels[i].ID < cfg.Channels[j].ID
		})
		sort.Slice(tc.expectedConfig.Channels, func(i, j int) bool {
			return tc.expectedConfig.Channels[i].ID < tc.expectedConfig.Channels[j].ID
		})
		assert.Equal(t, tc.expectedConfig, cfg, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.expectedConfig, cfg))
		repoCall.Unset()
	}
}

func TestUpdateConnections(t *testing.T) {
	svc, auth, sdk := newService()
	c := config

	ch := channel
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	created, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	c.ExternalID = testsutil.GenerateUUID(t)
	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 = sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 = sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
	active, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall1 = sdk.On("Connect", mock.Anything, mock.Anything).Return(nil)
	err = svc.ChangeState(context.Background(), validToken, active.ThingID, bootstrap.Active)
	assert.Nil(t, err, fmt.Sprintf("Changing state expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()

	nonExisting := config
	nonExisting.ThingID = unknown

	cases := []struct {
		desc        string
		token       string
		id          string
		connections []string
		err         error
	}{
		{
			desc:        "update connections for config with state Inactive",
			token:       validToken,
			id:          created.ThingID,
			connections: []string{ch.ID},
			err:         nil,
		},
		{
			desc:        "update connections for config with state Active",
			token:       validToken,
			id:          active.ThingID,
			connections: []string{ch.ID},
			err:         nil,
		},
		{
			desc:        "update connections for non-existing config",
			token:       validToken,
			id:          "",
			connections: []string{"3"},
			err:         svcerr.ErrNotFound,
		},
		{
			desc:        "update connections with invalid channels",
			token:       validToken,
			id:          created.ThingID,
			connections: []string{"wrong"},
			err:         svcerr.ErrNotFound,
		},
		{
			desc:        "update connections a config with wrong credentials",
			token:       invalidToken,
			id:          created.ThingKey,
			connections: []string{ch.ID, "3"},
			err:         svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
		err := svc.UpdateConnections(context.Background(), tc.token, tc.id, tc.connections)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestList(t *testing.T) {
	svc, auth, sdk := newService()
	numThings := 101
	var saved []bootstrap.Config
	for i := 0; i < numThings; i++ {
		c := config
		c.ExternalID = testsutil.GenerateUUID(t)
		c.ExternalKey = testsutil.GenerateUUID(t)
		c.Name = fmt.Sprintf("%s-%d", config.Name, i)
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(c.Channels[0]), nil)
		s, err := svc.Add(context.Background(), validToken, c)
		assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		saved = append(saved, s)
	}
	// Set one Thing to the different state
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall1 := sdk.On("Connect", mock.Anything, mock.Anything).Return(nil)
	err := svc.ChangeState(context.Background(), validToken, saved[41].ThingID, bootstrap.Active)
	assert.Nil(t, err, fmt.Sprintf("Changing config state expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()

	saved[41].State = bootstrap.Active

	cases := []struct {
		desc   string
		config bootstrap.ConfigsPage
		filter bootstrap.Filter
		offset uint64
		limit  uint64
		token  string
		err    error
	}{
		{
			desc: "list configs",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter: bootstrap.Filter{},
			token:  validToken,
			offset: 0,
			limit:  10,
			err:    nil,
		},
		{
			desc: "list configs with specified name",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: saved[95:96],
			},
			filter: bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:  validToken,
			offset: 0,
			limit:  100,
			err:    nil,
		},
		{
			desc:   "list configs with invalid token",
			config: bootstrap.ConfigsPage{},
			filter: bootstrap.Filter{},
			token:  invalidToken,
			offset: 0,
			limit:  10,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "list last page",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  95,
				Limit:   10,
				Configs: saved[95:],
			},
			filter: bootstrap.Filter{},
			token:  validToken,
			offset: 95,
			limit:  10,
			err:    nil,
		},
		{
			desc: "list configs with Active state",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter: bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			token:  validToken,
			offset: 35,
			limit:  20,
			err:    nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		result, err := svc.List(context.Background(), tc.token, tc.filter, tc.offset, tc.limit)
		assert.ElementsMatch(t, tc.config.Configs, result.Configs, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Configs, result.Configs))
		assert.Equal(t, tc.config.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Total, result.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRemove(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "view a config with wrong credentials",
			id:    saved.ThingID,
			token: invalidToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "remove an existing config",
			id:    saved.ThingID,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "remove removed config",
			id:    saved.ThingID,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "remove non-existing config",
			id:    unknown,
			token: validToken,
			err:   nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		err := svc.Remove(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestBootstrap(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	e, err := enc([]byte(saved.ExternalKey))
	assert.Nil(t, err, fmt.Sprintf("Encrypting external key expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		config      bootstrap.Config
		externalKey string
		externalID  string
		err         error
		encrypted   bool
	}{
		{
			desc:        "bootstrap using invalid external id",
			config:      bootstrap.Config{},
			externalID:  "invalid",
			externalKey: saved.ExternalKey,
			err:         svcerr.ErrNotFound,
			encrypted:   false,
		},
		{
			desc:        "bootstrap using invalid external key",
			config:      bootstrap.Config{},
			externalID:  saved.ExternalID,
			externalKey: "invalid",
			err:         bootstrap.ErrExternalKey,
			encrypted:   false,
		},
		{
			desc:        "bootstrap an existing config",
			config:      saved,
			externalID:  saved.ExternalID,
			externalKey: saved.ExternalKey,
			err:         nil,
			encrypted:   false,
		},
		{
			desc:        "bootstrap encrypted",
			config:      saved,
			externalID:  saved.ExternalID,
			externalKey: hex.EncodeToString(e),
			err:         nil,
			encrypted:   true,
		},
	}

	for _, tc := range cases {
		config, err := svc.Bootstrap(context.Background(), tc.externalKey, tc.externalID, tc.encrypted)
		assert.Equal(t, tc.config, config, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.config, config))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChangeState(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toGroup(config.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc  string
		state bootstrap.State
		id    string
		token string
		err   error
	}{
		{
			desc:  "change state with wrong credentials",
			state: bootstrap.Active,
			id:    saved.ThingID,
			token: invalidToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "change state of non-existing config",
			state: bootstrap.Active,
			id:    unknown,
			token: validToken,
			err:   svcerr.ErrNotFound,
		},
		{
			desc:  "change state to Active",
			state: bootstrap.Active,
			id:    saved.ThingID,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "change state to current state",
			state: bootstrap.Active,
			id:    saved.ThingID,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "change state to Inactive",
			state: bootstrap.Inactive,
			id:    saved.ThingID,
			token: validToken,
			err:   nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := sdk.On("Connect", mock.Anything, mock.Anything).Return(nil)
		repoCall2 := sdk.On("DisconnectThing", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := svc.ChangeState(context.Background(), tc.token, tc.id, tc.state)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	_, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
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
		err := svc.UpdateChannelHandler(context.Background(), tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveChannelHandler(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	_, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing channel",
			id:   channel.ID,
			err:  nil,
		},
		{
			desc: "remove a non-existing channel",
			id:   "unknown",
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveChannelHandler(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveCoinfigHandler(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing config",
			id:   saved.ThingID,
			err:  nil,
		},
		{
			desc: "remove a non-existing channel",
			id:   "unknown",
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveConfigHandler(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnectThingsHandler(t *testing.T) {
	svc, auth, sdk := newService()
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	cases := []struct {
		desc      string
		thingID   string
		channelID string
		err       error
	}{
		{
			desc:      "disconnect",
			channelID: channel.ID,
			thingID:   saved.ThingID,
			err:       nil,
		},
		{
			desc:      "disconnect disconnected",
			channelID: channel.ID,
			thingID:   saved.ThingID,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := svc.DisconnectThingHandler(context.Background(), tc.channelID, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func toGroup(ch bootstrap.Channel) mgsdk.Channel {
	return mgsdk.Channel{
		ID:       ch.ID,
		Name:     ch.Name,
		Metadata: ch.Metadata,
	}
}
