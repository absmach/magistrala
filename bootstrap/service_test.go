//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package bootstrap_test

import (
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
	mfsdk "github.com/mainflux/mainflux/sdk/go"
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
	unknownID    = "1"
	unknownKey   = "2"
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

func newService(users mainflux.UsersServiceClient, url string) bootstrap.Service {
	things := mocks.NewConfigsRepository(map[string]string{unknownID: unknownKey})
	config := mfsdk.Config{
		BaseURL: url,
	}

	sdk := mfsdk.NewSDK(config)
	return bootstrap.New(users, things, sdk, encKey)
}

func newThingsService(users mainflux.UsersServiceClient) things.Service {
	channels := make(map[string]things.Channel, channelsNum)
	for i := 0; i < channelsNum; i++ {
		id := strconv.Itoa(i + 1)
		channels[id] = things.Channel{
			ID:       id,
			Owner:    email,
			Metadata: map[string]interface{}{"meta": "data"},
		}
	}

	return mocks.NewThingsService(map[string]things.Thing{}, channels, users)
}

func newThingsServer(svc things.Service) *httptest.Server {
	mux := httpapi.MakeHandler(mocktracer.New(), svc)
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
	users := mocks.NewUsersService(map[string]string{validToken: email})

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
		key    string
		err    error
	}{
		{
			desc:   "add a new config",
			config: config,
			key:    validToken,
			err:    nil,
		},
		{
			desc:   "add a config with an invalid ID",
			config: neID,
			key:    validToken,
			err:    bootstrap.ErrNotFound,
		},
		{
			desc:   "add a config with wrong credentials",
			config: config,
			key:    invalidToken,
			err:    bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:   "add a config with invalid list of channels",
			config: wrongChannels,
			key:    validToken,
			err:    bootstrap.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := svc.Add(tc.key, tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestView(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc string
		id   string
		key  string
		err  error
	}{
		{
			desc: "view an existing config",
			id:   saved.MFThing,
			key:  validToken,
			err:  nil,
		},
		{
			desc: "view a non-existing config",
			id:   unknown,
			key:  validToken,
			err:  bootstrap.ErrNotFound,
		},
		{
			desc: "view a config with wrong credentials",
			id:   config.MFThing,
			key:  invalidToken,
			err:  bootstrap.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.View(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdate(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	saved, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	modifiedCreated := saved
	modifiedCreated.Content = "new-config"
	modifiedCreated.Name = "new name"

	nonExisting := config
	nonExisting.MFThing = unknown

	cases := []struct {
		desc   string
		config bootstrap.Config
		key    string
		err    error
	}{
		{
			desc:   "update a config with state Created",
			config: modifiedCreated,
			key:    validToken,
			err:    nil,
		},
		{
			desc:   "update a non-existing config",
			config: nonExisting,
			key:    validToken,
			err:    bootstrap.ErrNotFound,
		},
		{
			desc:   "update a config with wrong credentials",
			config: saved,
			key:    invalidToken,
			err:    bootstrap.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.Update(tc.key, tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateCert(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	saved, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc       string
		key        string
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
			key:        validToken,
			err:        nil,
		},
		{
			desc:       "update cert for a non-existing config",
			thingKey:   "empty",
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",

			key: validToken,
			err: bootstrap.ErrNotFound,
		},
		{
			desc:       "update config cert with wrong credentials",
			thingKey:   saved.MFKey,
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",
			key:        invalidToken,
			err:        bootstrap.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateCert(tc.key, tc.thingKey, tc.clientCert, tc.clientKey, tc.caCert)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateConnections(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	created, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	externalID, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ExternalID = externalID.String()
	active, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	err = svc.ChangeState(validToken, active.MFThing, bootstrap.Active)
	require.Nil(t, err, fmt.Sprintf("Changing state expected to succeed: %s.\n", err))

	nonExisting := config
	nonExisting.MFThing = unknown

	cases := []struct {
		desc        string
		key         string
		id          string
		connections []string
		err         error
	}{
		{
			desc:        "update connections for config with state Inactive",
			key:         validToken,
			id:          created.MFThing,
			connections: []string{"2"},
			err:         nil,
		},
		{
			desc:        "update connections for config with state Active",
			key:         validToken,
			id:          active.MFThing,
			connections: []string{"3"},
			err:         nil,
		},
		{
			desc:        "update connections for non-existing config",
			key:         validToken,
			id:          "",
			connections: []string{"3"},
			err:         bootstrap.ErrNotFound,
		},
		{
			desc:        "update connections with invalid channels",
			key:         validToken,
			id:          created.MFThing,
			connections: []string{"wrong"},
			err:         bootstrap.ErrMalformedEntity,
		},
		{
			desc:        "update connections a config with wrong credentials",
			key:         invalidToken,
			id:          created.MFKey,
			connections: []string{"2", "3"},
			err:         bootstrap.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateConnections(tc.key, tc.id, tc.connections)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestList(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

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
		s, err := svc.Add(validToken, c)
		saved = append(saved, s)
		require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	}
	// Set one Thing to the different state
	err := svc.ChangeState(validToken, "42", bootstrap.Active)
	require.Nil(t, err, fmt.Sprintf("Changing config state expected to succeed: %s.\n", err))
	saved[41].State = bootstrap.Active

	unknownConfig := bootstrap.Config{
		ExternalID:  unknownID,
		ExternalKey: unknownKey,
	}

	cases := []struct {
		desc   string
		config bootstrap.ConfigsPage
		filter bootstrap.Filter
		offset uint64
		limit  uint64
		key    string
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
			key:    validToken,
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
			key:    validToken,
			offset: 0,
			limit:  100,
			err:    nil,
		},
		{
			desc:   "list configs unauthorized",
			config: bootstrap.ConfigsPage{},
			filter: bootstrap.Filter{},
			key:    invalidToken,
			offset: 0,
			limit:  10,
			err:    bootstrap.ErrUnauthorizedAccess,
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
			key:    validToken,
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
			key:    validToken,
			offset: 35,
			limit:  20,
			err:    nil,
		},
		{
			desc: "list unknown configs",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   20,
				Configs: []bootstrap.Config{unknownConfig},
			},
			filter: bootstrap.Filter{Unknown: true},
			key:    validToken,
			offset: 0,
			limit:  20,
			err:    nil,
		},
	}

	for _, tc := range cases {
		result, err := svc.List(tc.key, tc.filter, tc.offset, tc.limit)
		assert.ElementsMatch(t, tc.config.Configs, result.Configs, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Configs, result.Configs))
		assert.Equal(t, tc.config.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Total, result.Total))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc string
		id   string
		key  string
		err  error
	}{
		{
			desc: "view a config with wrong credentials",
			id:   saved.MFThing,
			key:  invalidToken,
			err:  bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc: "remove an existing config",
			id:   saved.MFThing,
			key:  validToken,
			err:  nil,
		},
		{
			desc: "remove removed config",
			id:   saved.MFThing,
			key:  validToken,
			err:  nil,
		},
		{
			desc: "remove non-existing config",
			id:   unknown,
			key:  validToken,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := svc.Remove(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestBootstrap(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(validToken, config)
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
			err:         bootstrap.ErrNotFound,
			encrypted:   false,
		},
		{
			desc:        "bootstrap using invalid external key",
			config:      bootstrap.Config{},
			externalID:  saved.ExternalID,
			externalKey: "invalid",
			err:         bootstrap.ErrNotFound,
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
		config, err := svc.Bootstrap(tc.externalKey, tc.externalID, tc.encrypted)
		assert.Equal(t, tc.config, config, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.config, config))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChangeState(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc  string
		state bootstrap.State
		id    string
		key   string
		err   error
	}{
		{
			desc:  "change state with wrong credentials",
			state: bootstrap.Active,
			id:    saved.MFThing,
			key:   invalidToken,
			err:   bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:  "change state of non-existing config",
			state: bootstrap.Active,
			id:    unknown,
			key:   validToken,
			err:   bootstrap.ErrNotFound,
		},
		{
			desc:  "change state to Active",
			state: bootstrap.Active,
			id:    saved.MFThing,
			key:   validToken,
			err:   nil,
		},
		{
			desc:  "change state to current state",
			state: bootstrap.Active,
			id:    saved.MFThing,
			key:   validToken,
			err:   nil,
		},
		{
			desc:  "change state to Inactive",
			state: bootstrap.Inactive,
			id:    saved.MFThing,
			key:   validToken,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.ChangeState(tc.key, tc.id, tc.state)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	_, err := svc.Add(validToken, config)
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
		err := svc.UpdateChannelHandler(tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveChannelHandler(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	_, err := svc.Add(validToken, config)
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
		err := svc.RemoveChannelHandler(tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveCoinfigHandler(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(validToken, config)
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
		err := svc.RemoveConfigHandler(tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnectThingsHandler(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(validToken, config)
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
		err := svc.DisconnectThingHandler(tc.channelID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
