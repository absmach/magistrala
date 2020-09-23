// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	grpcapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const wrongID = ""

var (
	thing   = things.Thing{Name: "test_app", Metadata: map[string]interface{}{"test": "test"}}
	channel = things.Channel{Name: "test", Metadata: map[string]interface{}{"test": "test"}}
)

func TestCanAccessByKey(t *testing.T) {
	ths, err := svc.CreateThings(context.Background(), token, thing, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1 := ths[0]
	th2 := ths[1]

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th1.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(usersAddr, grpc.WithInsecure())
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		key     string
		chanID  string
		thingID string
		code    codes.Code
	}{
		"check if connected thing can access existing channel": {
			key:     th1.Key,
			chanID:  ch.ID,
			thingID: th1.ID,
			code:    codes.OK,
		},
		"check if unconnected thing can access existing channel": {
			key:     th2.Key,
			chanID:  ch.ID,
			thingID: wrongID,
			code:    codes.PermissionDenied,
		},
		"check if thing with wrong access key can access existing channel": {
			key:     wrong,
			chanID:  ch.ID,
			thingID: wrongID,
			code:    codes.NotFound,
		},
		"check if connected thing can access non-existent channel": {
			key:     th1.Key,
			chanID:  wrongID,
			thingID: wrongID,
			code:    codes.InvalidArgument,
		},
	}

	for desc, tc := range cases {
		id, err := cli.CanAccessByKey(ctx, &mainflux.AccessByKeyReq{Token: tc.key, ChanID: tc.chanID})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.thingID, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.thingID, id.GetValue()))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}

func TestCanAccessByID(t *testing.T) {
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1 := ths[0]
	ths, err = svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th2 := ths[0]

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	svc.Connect(context.Background(), token, []string{ch.ID}, []string{th2.ID})

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(usersAddr, grpc.WithInsecure())
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		chanID  string
		thingID string
		code    codes.Code
	}{
		"check if connected thing can access existing channel": {
			chanID:  ch.ID,
			thingID: th2.ID,
			code:    codes.OK,
		},
		"check if unconnected thing can access existing channel": {
			chanID:  ch.ID,
			thingID: th1.ID,
			code:    codes.PermissionDenied,
		},
		"check if connected thing can access non-existent channel": {
			chanID:  wrongID,
			thingID: th2.ID,
			code:    codes.InvalidArgument,
		},
		"check if thing with empty ID can access existing channel": {
			chanID:  ch.ID,
			thingID: "",
			code:    codes.InvalidArgument,
		},
		"check if connected thing can access channel with empty ID": {
			chanID:  "",
			thingID: th2.ID,
			code:    codes.InvalidArgument,
		},
	}

	for desc, tc := range cases {
		_, err := cli.CanAccessByID(ctx, &mainflux.AccessByIDReq{ThingID: tc.thingID, ChanID: tc.chanID})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}

func TestIdentify(t *testing.T) {
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	sth := ths[0]

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(usersAddr, grpc.WithInsecure())
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		key  string
		id   string
		code codes.Code
	}{
		"identify existing thing": {
			key:  sth.Key,
			id:   sth.ID,
			code: codes.OK,
		},
		"identify non-existent thing": {
			key:  wrong,
			id:   wrongID,
			code: codes.NotFound,
		},
	}

	for desc, tc := range cases {
		id, err := cli.Identify(ctx, &mainflux.Token{Value: tc.key})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}
