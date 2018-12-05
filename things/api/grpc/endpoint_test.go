//
// Copyright (c) 2018
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

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	grpcapi "github.com/mainflux/mainflux/things/api/grpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const wrongID = ""

var (
	thing   = things.Thing{Type: "app", Name: "test_app", Metadata: "test_metadata"}
	channel = things.Channel{Name: "test"}
)

func TestCanAccess(t *testing.T) {
	oth, _ := svc.AddThing(token, thing)
	cth, _ := svc.AddThing(token, thing)
	sch, _ := svc.CreateChannel(token, channel)
	svc.Connect(token, sch.ID, cth.ID)

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(usersAddr, grpc.WithInsecure())
	cli := grpcapi.NewClient(conn)
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

func TestIdentify(t *testing.T) {
	sth, _ := svc.AddThing(token, thing)

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(usersAddr, grpc.WithInsecure())
	cli := grpcapi.NewClient(conn)
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
