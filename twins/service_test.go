// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package twins_test

import (
	"context"
	"fmt"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/twins"
	"github.com/mainflux/mainflux/twins/mocks"
	nats "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	twmqtt "github.com/mainflux/mainflux/twins/mqtt"
)

const (
	twinName   = "name"
	wrongID    = ""
	token      = "token"
	wrongToken = "wrong-token"
	email      = "user@example.com"
	natsURL    = "nats://localhost:4222"
	mqttURL    = "tcp://localhost:1883"
	topic      = "topic"
)

func newService(tokens map[string]string) twins.Service {
	auth := mocks.NewAuthNServiceClient(tokens)
	twinsRepo := mocks.NewTwinRepository()
	statesRepo := mocks.NewStateRepository()
	idp := mocks.NewIdentityProvider()

	nc, _ := nats.Connect(natsURL)

	opts := mqtt.NewClientOptions()
	pc := mqtt.NewClient(opts)

	mc := twmqtt.New(pc, topic)

	return twins.New(nc, mc, auth, twinsRepo, statesRepo, idp)
}

func TestAddTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	twin := twins.Twin{}
	def := twins.Definition{}

	cases := []struct {
		desc  string
		twin  twins.Twin
		token string
		err   error
	}{
		{
			desc:  "add new twin",
			twin:  twin,
			token: token,
			err:   nil,
		},
		{
			desc:  "add twin with wrong credentials",
			twin:  twin,
			token: wrongToken,
			err:   twins.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.AddTwin(context.Background(), tc.token, tc.twin, def)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	twin := twins.Twin{}
	other := twins.Twin{}
	def := twins.Definition{}

	other.ID = wrongID
	saved, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	saved.Name = twinName

	cases := []struct {
		desc  string
		twin  twins.Twin
		token string
		err   error
	}{
		{
			desc:  "update existing twin",
			twin:  saved,
			token: token,
			err:   nil,
		},
		{
			desc:  "update twin with wrong credentials",
			twin:  saved,
			token: wrongToken,
			err:   twins.ErrUnauthorizedAccess,
		},
		{
			desc:  "update non-existing twin",
			twin:  other,
			token: token,
			err:   twins.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateTwin(context.Background(), tc.token, tc.twin, def)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	twin := twins.Twin{}
	def := twins.Definition{}
	saved, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing twin": {
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		"view twin with wrong credentials": {
			id:    saved.ID,
			token: wrongToken,
			err:   twins.ErrUnauthorizedAccess,
		},
		"view non-existing twin": {
			id:    wrongID,
			token: token,
			err:   twins.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewTwin(context.Background(), tc.token, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListTwins(t *testing.T) {
	svc := newService(map[string]string{token: email})
	twin := twins.Twin{Name: twinName, Owner: email}
	def := twins.Definition{}
	m := make(map[string]interface{})
	m["serial"] = "123456"
	twin.Metadata = m

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		svc.AddTwin(context.Background(), token, twin, def)
	}

	cases := map[string]struct {
		token    string
		offset   uint64
		limit    uint64
		size     uint64
		metadata map[string]interface{}
		err      error
	}{
		"list all twins": {
			token:  token,
			offset: 0,
			limit:  n,
			size:   n,
			err:    nil,
		},
		"list with zero limit": {
			token:  token,
			limit:  0,
			offset: 0,
			size:   0,
			err:    nil,
		},
		"list with offset and limit": {
			token:  token,
			offset: 8,
			limit:  5,
			size:   2,
			err:    nil,
		},
		"list with wrong credentials": {
			token:  wrongToken,
			limit:  0,
			offset: n,
			err:    twins.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListTwins(context.Background(), tc.token, tc.offset, tc.limit, twinName, tc.metadata)
		size := uint64(len(page.Twins))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	twin := twins.Twin{}
	def := twins.Definition{}
	saved, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove twin with wrong credentials",
			id:    saved.ID,
			token: wrongToken,
			err:   twins.ErrUnauthorizedAccess,
		},
		{
			desc:  "remove existing twin",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed twin",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove non-existing twin",
			id:    wrongID,
			token: token,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveTwin(context.Background(), tc.token, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
