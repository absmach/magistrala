// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	authv1 "github.com/absmach/fluxmq/pkg/proto/auth/v1"
	"github.com/absmach/fluxmq/pkg/proto/auth/v1/authv1connect"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauth "github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

var _ authv1connect.AuthServiceHandler = (*connectServer)(nil)

type connectServer struct {
	authv1connect.UnimplementedAuthServiceHandler
	clients  grpcClientsV1.ClientsServiceClient
	channels grpcChannelsV1.ChannelsServiceClient
	parser   messaging.TopicParser
}

// NewServer creates a FluxMQ AuthService Connect handler that bridges to
// SuperMQ's Clients (authn) and Channels (authz) services.
func NewServer(
	clients grpcClientsV1.ClientsServiceClient,
	channels grpcChannelsV1.ChannelsServiceClient,
	parser messaging.TopicParser,
) authv1connect.AuthServiceHandler {
	return &connectServer{
		clients:  clients,
		channels: channels,
		parser:   parser,
	}
}

func (s *connectServer) Authenticate(ctx context.Context, req *connect.Request[authv1.AuthnReq]) (*connect.Response[authv1.AuthnRes], error) {
	username := req.Msg.GetUsername()
	password := req.Msg.GetPassword()

	token := authn.AuthPack(authn.BasicAuth, username, password)
	res, err := s.clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{Token: token})
	if err != nil {
		if !shouldTryDomainAuth(req.Msg, username, password) {
			return nil, encodeError(err)
		}

		token = authn.AuthPack(authn.DomainAuth, username, password)
		res, err = s.clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{Token: token})
		if err != nil {
			return nil, encodeError(err)
		}
	}

	return connect.NewResponse(&authv1.AuthnRes{
		Authenticated: res.GetAuthenticated(),
		Id:            res.GetId(),
	}), nil
}

func (s *connectServer) Authorize(ctx context.Context, req *connect.Request[authv1.AuthzReq]) (*connect.Response[authv1.AuthzRes], error) {
	connType := connections.ConnType(req.Msg.GetAction())
	if err := connections.CheckConnType(connType); err != nil {
		return nil, encodeError(err)
	}

	var domainID, channelID string
	var topicType messaging.TopicType
	var err error

	switch connType {
	case connections.Publish:
		domainID, channelID, _, topicType, err = s.parser.ParsePublishTopic(ctx, req.Msg.GetTopic(), true)
	case connections.Subscribe:
		domainID, channelID, _, topicType, err = s.parser.ParseSubscribeTopic(ctx, req.Msg.GetTopic(), true)
	}
	if err != nil {
		if shouldDenyAuthorize(err) {
			return connect.NewResponse(&authv1.AuthzRes{Authorized: false}), nil
		}
		return nil, encodeError(err)
	}

	if topicType == messaging.HealthType {
		return connect.NewResponse(&authv1.AuthzRes{Authorized: true}), nil
	}

	ar := &grpcChannelsV1.AuthzReq{
		Type:       uint32(connType),
		ClientId:   req.Msg.GetExternalId(),
		ClientType: policies.ClientType,
		ChannelId:  channelID,
		DomainId:   domainID,
	}
	res, err := s.channels.Authorize(ctx, ar)
	if err != nil {
		if shouldDenyAuthorize(err) {
			return connect.NewResponse(&authv1.AuthzRes{Authorized: false}), nil
		}
		return nil, encodeError(err)
	}

	return connect.NewResponse(&authv1.AuthzRes{
		Authorized: res.GetAuthorized(),
	}), nil
}

func shouldTryDomainAuth(msg *authv1.AuthnReq, username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	return strings.HasPrefix(msg.GetClientId(), "http:")
}

func shouldDenyAuthorize(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, svcerr.ErrNotFound),
		errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, messaging.ErrMalformedTopic),
		err == apiutil.ErrMissingID:
		return true
	}

	// Backward compatibility for gRPC client layers that may return
	// Internal with a payload containing "entity not found".
	return strings.Contains(err.Error(), svcerr.ErrNotFound.Error())
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, smqauth.ErrKeyExpired):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Contains(err, svcerr.ErrAuthorization):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Contains(err, messaging.ErrMalformedTopic):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
