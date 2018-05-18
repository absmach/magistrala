package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	grpcapi "github.com/mainflux/mainflux/things/api/grpc"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	port  = 8080
	token = "token"
	wrong = "wrong"
	email = "john.doe@email.com"
)

var (
	thing   = things.Thing{Type: "app", Name: "test_app", Payload: "test_payload"}
	channel = things.Channel{Name: "test"}
	svc     things.Service
)

func newService(tokens map[string]string) things.Service {
	users := mocks.NewUsersService(tokens)
	thingsRepo := mocks.NewThingRepository()
	channelsRepo := mocks.NewChannelRepository(thingsRepo)
	idp := mocks.NewIdentityProvider()
	return things.New(users, thingsRepo, channelsRepo, idp)
}

func startGRPCServer(svc things.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	mainflux.RegisterThingsServiceServer(server, grpcapi.NewServer(svc))
	go server.Serve(listener)
}

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
		thingKey string
		chanID   string
		id       string
		code     codes.Code
	}{
		"check if connected thing can access existing channel":             {cth.Key, sch.ID, cth.ID, codes.OK},
		"check if unconnected thing can access existing channel":           {oth.Key, sch.ID, "", codes.PermissionDenied},
		"check if thing with wrong access key can access existing channel": {wrong, sch.ID, "", codes.PermissionDenied},
		"check if connected thing can access non-existent channel":         {cth.Key, wrong, "", codes.InvalidArgument},
	}

	for desc, tc := range cases {
		id, err := cli.CanAccess(ctx, &mainflux.AccessReq{tc.thingKey, tc.chanID})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id.GetValue()))
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
		"identify existing thing":     {sth.Key, sth.ID, codes.OK},
		"identify non-existent thing": {wrong, "", codes.PermissionDenied},
	}

	for desc, tc := range cases {
		id, err := cli.Identify(ctx, &mainflux.Token{Value: tc.key})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}
