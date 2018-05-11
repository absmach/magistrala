package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/clients"
	grpcapi "github.com/mainflux/mainflux/clients/api/grpc"
	"github.com/mainflux/mainflux/clients/mocks"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	port  = 8080
	token = "token"
	email = "john.doe@email.com"
)

var (
	client  = clients.Client{Type: "app", Name: "test_app", Payload: "test_payload"}
	channel = clients.Channel{Name: "test"}
)

func newService(tokens map[string]string) clients.Service {
	users := mocks.NewUsersService(tokens)
	clientsRepo := mocks.NewClientRepository()
	channelsRepo := mocks.NewChannelRepository(clientsRepo)
	hasher := mocks.NewHasher()
	idp := mocks.NewIdentityProvider()

	return clients.New(users, clientsRepo, channelsRepo, hasher, idp)
}

func startGRPCServer(svc clients.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	mainflux.RegisterClientsServiceServer(server, grpcapi.NewServer(svc))
	go server.Serve(listener)
}

func TestCanAccess(t *testing.T) {
	svc := newService(map[string]string{token: email})
	startGRPCServer(svc, port)

	connectedClientID, _ := svc.AddClient(token, client)
	connectedClient, _ := svc.ViewClient(token, connectedClientID)

	clientID, _ := svc.AddClient(token, client)
	client, _ := svc.ViewClient(token, clientID)

	chanID, _ := svc.CreateChannel(token, channel)
	svc.Connect(token, chanID, connectedClientID)

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(usersAddr, grpc.WithInsecure())
	cli := grpcapi.NewClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		clientKey string
		chanID    string
		id        string
		code      codes.Code
	}{
		"check if connected client can access existing channel":     {connectedClient.Key, chanID, connectedClientID, codes.OK},
		"check if unconnected client can access existing channel":   {client.Key, chanID, "", codes.PermissionDenied},
		"check if wrong client can access existing channel":         {mocks.UnauthorizedToken, chanID, "", codes.Unauthenticated},
		"check if connected client can access non-existent channel": {connectedClient.Key, "1", "", codes.InvalidArgument},
	}

	for desc, tc := range cases {
		id, err := cli.CanAccess(ctx, &mainflux.AccessReq{tc.clientKey, tc.chanID})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}
