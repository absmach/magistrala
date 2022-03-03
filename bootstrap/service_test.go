// Copyright (c) Mainflux
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
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/gofrs/uuid"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/things/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validToken   = "validToken"
	invalidToken = "invalidToken"
	email        = "test@example.com"
	unknown      = "unknown"
	channelsNum  = 3
)

var (
	encKey = []byte("1234567891011121")

	channel = bootstrap.Channel{
		ID:       "1",
		Name:     "name",
		Metadata: map[string]interface{}{"name": "value"},
	}

	config = bootstrap.Config{
		ExternalID:  "external_id",
		ExternalKey: "external_key",
		MFChannels:  []bootstrap.Channel{channel},
		Content:     "config",
	}
)

func newService(auth mainflux.AuthServiceClient, url string) bootstrap.Service {
	things := mocks.NewConfigsRepository()
	config := mfsdk.Config{
		ThingsURL: url,
	}

	sdk := mfsdk.NewSDK(config)
	return bootstrap.New(auth, things, sdk, encKey)
}

func newThingsService(auth mainflux.AuthServiceClient) things.Service {
	channels := make(map[string]things.Channel, channelsNum)
	for i := 0; i < channelsNum; i++ {
		id := strconv.Itoa(i + 1)
		channels[id] = things.Channel{
			ID:       id,
			Owner:    email,
			Metadata: map[string]interface{}{"meta": "data"},
		}
	}

	return mocks.NewThingsService(map[string]things.Thing{}, channels, auth)
}

func newThingsServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
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
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	neID := config
	neID.MFThing = "non-existent"

	wrongChannels := config
	ch := channel
	ch.ID = "invalid"
	wrongChannels.MFChannels = append(wrongChannels.MFChannels, ch)

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
			err:    errors.ErrNotFound,
		},
		{
			desc:   "add a config with wrong credentials",
			config: config,
			token:  invalidToken,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "add a config with invalid list of channels",
			config: wrongChannels,
			token:  validToken,
			err:    errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := svc.Add(context.Background(), tc.token, tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestView(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "view an existing config",
			id:    saved.MFThing,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "view a non-existing config",
			id:    unknown,
			token: validToken,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view a config with wrong credentials",
			id:    config.MFThing,
			token: invalidToken,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		_, err := svc.View(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdate(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	modifiedCreated := saved
	modifiedCreated.Content = "new-config"
	modifiedCreated.Name = "new name"

	nonExisting := config
	nonExisting.MFThing = unknown

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
			err:    errors.ErrNotFound,
		},
		{
			desc:   "update a config with wrong credentials",
			config: saved,
			token:  invalidToken,
			err:    errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.Update(context.Background(), tc.token, tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateCert(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc       string
		token      string
		thingKey   string
		clientCert string
		clientKey  string
		caCert     string
		err        error
	}{
		{
			desc:       "update certs for the valid config",
			thingKey:   saved.MFKey,
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",
			token:      validToken,
			err:        nil,
		},
		{
			desc:       "update cert for a non-existing config",
			thingKey:   "empty",
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",

			token: validToken,
			err:   errors.ErrNotFound,
		},
		{
			desc:       "update config cert with wrong credentials",
			thingKey:   saved.MFKey,
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",
			token:      invalidToken,
			err:        errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateCert(context.Background(), tc.token, tc.thingKey, tc.clientCert, tc.clientKey, tc.caCert)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateConnections(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	created, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	externalID, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ExternalID = externalID.String()
	active, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	err = svc.ChangeState(context.Background(), validToken, active.MFThing, bootstrap.Active)
	require.Nil(t, err, fmt.Sprintf("Changing state expected to succeed: %s.\n", err))

	nonExisting := config
	nonExisting.MFThing = unknown

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
			id:          created.MFThing,
			connections: []string{"2"},
			err:         nil,
		},
		{
			desc:        "update connections for config with state Active",
			token:       validToken,
			id:          active.MFThing,
			connections: []string{"3"},
			err:         nil,
		},
		{
			desc:        "update connections for non-existing config",
			token:       validToken,
			id:          "",
			connections: []string{"3"},
			err:         errors.ErrNotFound,
		},
		{
			desc:        "update connections with invalid channels",
			token:       validToken,
			id:          created.MFThing,
			connections: []string{"wrong"},
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "update connections a config with wrong credentials",
			token:       invalidToken,
			id:          created.MFKey,
			connections: []string{"2", "3"},
			err:         errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateConnections(context.Background(), tc.token, tc.id, tc.connections)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestList(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	numThings := 101
	var saved []bootstrap.Config
	for i := 0; i < numThings; i++ {
		c := config
		id, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		c.ExternalID = id.String()
		c.ExternalKey = id.String()
		c.Name = fmt.Sprintf("%s-%d", config.Name, i)
		s, err := svc.Add(context.Background(), validToken, c)
		saved = append(saved, s)
		require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	}
	// Set one Thing to the different state
	err := svc.ChangeState(context.Background(), validToken, "42", bootstrap.Active)
	require.Nil(t, err, fmt.Sprintf("Changing config state expected to succeed: %s.\n", err))
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
			err:    errors.ErrAuthentication,
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
		result, err := svc.List(context.Background(), tc.token, tc.filter, tc.offset, tc.limit)
		assert.ElementsMatch(t, tc.config.Configs, result.Configs, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Configs, result.Configs))
		assert.Equal(t, tc.config.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Total, result.Total))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "view a config with wrong credentials",
			id:    saved.MFThing,
			token: invalidToken,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove an existing config",
			id:    saved.MFThing,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "remove removed config",
			id:    saved.MFThing,
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
		err := svc.Remove(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestBootstrap(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	e, err := enc([]byte(saved.ExternalKey))
	require.Nil(t, err, fmt.Sprintf("Encrypting external key expected to succeed: %s.\n", err))

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
			err:         errors.ErrNotFound,
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
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

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
			id:    saved.MFThing,
			token: invalidToken,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "change state of non-existing config",
			state: bootstrap.Active,
			id:    unknown,
			token: validToken,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "change state to Active",
			state: bootstrap.Active,
			id:    saved.MFThing,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "change state to current state",
			state: bootstrap.Active,
			id:    saved.MFThing,
			token: validToken,
			err:   nil,
		},
		{
			desc:  "change state to Inactive",
			state: bootstrap.Inactive,
			id:    saved.MFThing,
			token: validToken,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.ChangeState(context.Background(), tc.token, tc.id, tc.state)
		assert.True(t, errors.Contains(err, tc.err), err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	_, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
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
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	_, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

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
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing config",
			id:   saved.MFThing,
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
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc      string
		thingID   string
		channelID string
		err       error
	}{
		{
			desc:      "disconnect",
			channelID: channel.ID,
			thingID:   saved.MFThing,
			err:       nil,
		},
		{
			desc:      "disconnect disconnected",
			channelID: channel.ID,
			thingID:   saved.MFThing,
			err:       nil,
		},
	}

	for _, tc := range cases {
		err := svc.DisconnectThingHandler(context.Background(), tc.channelID, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
