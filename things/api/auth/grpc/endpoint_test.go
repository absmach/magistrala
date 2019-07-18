//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const wrongID = ""

var (
	thing   = things.Thing{Name: "test_app", Metadata: map[string]interface{}{"test": "test"}}
	channel = things.Channel{Name: "test", Metadata: map[string]interface{}{"test": "test"}}
)

func TestCanAccess(t *testing.T) {
	oth, _ := svc.AddThing(context.Background(), token, thing)
	cth, _ := svc.AddThing(context.Background(), token, thing)
	sch, _ := svc.CreateChannel(context.Background(), token, channel)
	svc.Connect(context.Background(), token, sch.ID, cth.ID)

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(usersAddr, grpc.WithInsecure())
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
			key:     cth.Key,
			chanID:  sch.ID,
			thingID: cth.ID,
			code:    codes.OK,
		},
		"check if unconnected thing can access existing channel": {
			key:     oth.Key,
			chanID:  sch.ID,
			thingID: wrongID,
			code:    codes.PermissionDenied,
		},
		"check if thing with wrong access key can access existing channel": {
			key:     wrong,
			chanID:  sch.ID,
			thingID: wrongID,
			code:    codes.PermissionDenied,
		},
		"check if connected thing can access non-existent channel": {
			key:     cth.Key,
			chanID:  wrongID,
			thingID: wrongID,
			code:    codes.InvalidArgument,
		},
	}

	for desc, tc := range cases {
		id, err := cli.CanAccess(ctx, &mainflux.AccessReq{Token: tc.key, ChanID: tc.chanID})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.thingID, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.thingID, id.GetValue()))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}

func TestCanAccessByID(t *testing.T) {
	oth, _ := svc.AddThing(context.Background(), token, thing)
	cth, _ := svc.AddThing(context.Background(), token, thing)
	sch, _ := svc.CreateChannel(context.Background(), token, channel)
	svc.Connect(context.Background(), token, sch.ID, cth.ID)

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(usersAddr, grpc.WithInsecure())
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		chanID  string
		thingID string
		code    codes.Code
	}{
		"check if connected thing can access existing channel": {
			chanID:  sch.ID,
			thingID: cth.ID,
			code:    codes.OK,
		},
		"check if unconnected thing can access existing channel": {
			chanID:  sch.ID,
			thingID: oth.ID,
			code:    codes.PermissionDenied,
		},
		"check if connected thing can access non-existent channel": {
			chanID:  wrongID,
			thingID: cth.ID,
			code:    codes.InvalidArgument,
		},
		"check if thing with empty ID can access existing channel": {
			chanID:  sch.ID,
			thingID: "",
			code:    codes.InvalidArgument,
		},
		"check if connected thing can access channel with empty ID": {
			chanID:  "",
			thingID: cth.ID,
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
	sth, _ := svc.AddThing(context.Background(), token, thing)

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(usersAddr, grpc.WithInsecure())
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
			code: codes.PermissionDenied,
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
