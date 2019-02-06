//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package bootstrap_test

import (
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/mocks"
	mfsdk "github.com/mainflux/mainflux/sdk/go"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/http"
	uuid "github.com/satori/go.uuid"
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
	channel = bootstrap.Channel{
		ID:       "1",
		Name:     "name",
		Metadata: `{"name":"value"}`,
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
	return bootstrap.New(users, things, sdk)
}

func newThingsService(users mainflux.UsersServiceClient) things.Service {
	channels := make(map[string]things.Channel, channelsNum)
	for i := 0; i < channelsNum; i++ {
		id := strconv.Itoa(i + 1)
		channels[id] = things.Channel{
			ID:       id,
			Owner:    email,
			Metadata: `{"meta":"data"}`,
		}
	}

	return mocks.NewThingsService(map[string]things.Thing{}, channels, users)
}

func newThingsServer(svc things.Service) *httptest.Server {
	mux := httpapi.MakeHandler(svc)
	return httptest.NewServer(mux)
}

func TestAdd(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

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
	ch.ID = "3"
	modifiedCreated.MFChannels = []bootstrap.Channel{channel, ch}
	modifiedCreated.State = bootstrap.Active

	modifiedActive := modifiedCreated
	ch.ID = "2"
	modifiedActive.MFChannels = []bootstrap.Channel{channel, ch}

	nonExisting := config
	nonExisting.MFThing = unknown

	wrongChannels := modifiedActive
	ch = channel
	ch.ID = unknown
	wrongChannels.MFChannels = append(wrongChannels.MFChannels, ch)

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
			desc:   "update a config with state Active",
			config: modifiedActive,
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
		{
			desc:   "update a config with invalid list of channels",
			config: wrongChannels,
			key:    validToken,
			err:    bootstrap.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := svc.Update(tc.key, tc.config)
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
		id := uuid.NewV4().String()
		c.ExternalID = id
		c.ExternalKey = id
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
		config []bootstrap.Config
		filter bootstrap.Filter
		offset uint64
		limit  uint64
		key    string
		err    error
	}{
		{
			desc:   "list configs",
			config: saved[0:10],
			filter: bootstrap.Filter{},
			key:    validToken,
			offset: 0,
			limit:  10,
			err:    nil,
		},
		{
			desc:   "list configs with specified name",
			config: saved[95:96],
			filter: bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			key:    validToken,
			offset: 0,
			limit:  100,
			err:    nil,
		},
		{
			desc:   "list configs unauthorized",
			config: []bootstrap.Config{},
			filter: bootstrap.Filter{},
			key:    invalidToken,
			offset: 0,
			limit:  10,
			err:    bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:   "list last page",
			config: saved[95:],
			filter: bootstrap.Filter{},
			key:    validToken,
			offset: 95,
			limit:  10,
			err:    nil,
		},
		{
			desc:   "list configs with Active staate",
			config: []bootstrap.Config{saved[41]},
			filter: bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			key:    validToken,
			offset: 35,
			limit:  20,
			err:    nil,
		},
		{
			desc:   "list unknown configs",
			config: []bootstrap.Config{unknownConfig},
			filter: bootstrap.Filter{Unknown: true},
			key:    validToken,
			offset: 0,
			limit:  20,
			err:    nil,
		},
	}

	for _, tc := range cases {
		result, err := svc.List(tc.key, tc.filter, tc.offset, tc.limit)
		assert.ElementsMatch(t, tc.config, result, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config, result))
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

	cases := []struct {
		desc        string
		config      bootstrap.Config
		externalKey string
		externalID  string
		err         error
	}{
		{
			desc:        "bootstrap using invalid external id",
			config:      bootstrap.Config{},
			externalID:  "invalid",
			externalKey: saved.ExternalKey,
			err:         bootstrap.ErrNotFound,
		},
		{
			desc:        "bootstrap using invalid external key",
			config:      bootstrap.Config{},
			externalID:  saved.ExternalID,
			externalKey: "invalid",
			err:         bootstrap.ErrNotFound,
		},
		{
			desc:        "bootstrap an existing config",
			config:      saved,
			externalID:  saved.ExternalID,
			externalKey: saved.ExternalKey,
			err:         nil,
		},
	}

	for _, tc := range cases {
		config, err := svc.Bootstrap(tc.externalKey, tc.externalID)
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
