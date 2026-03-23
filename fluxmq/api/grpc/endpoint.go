// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/go-kit/kit/endpoint"
)

func authenticateEndpoint(clients grpcClientsV1.ClientsServiceClient) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(authenticateReq)

		token := authn.AuthPack(authn.BasicAuth, req.username, req.password)
		res, err := clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{Token: token})
		if err != nil {
			return authenticateRes{}, err
		}
		if !res.GetAuthenticated() {
			return authenticateRes{}, nil
		}

		return authenticateRes{
			authenticated: true,
			id:            res.GetId(),
		}, nil
	}
}

func authorizeEndpoint(channels grpcChannelsV1.ChannelsServiceClient, parser messaging.TopicParser) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(authorizeReq)

		connType := connections.ConnType(req.action)
		if err := connections.CheckConnType(connType); err != nil {
			return authorizeRes{}, err
		}

		var domainID, channelID string
		var topicType messaging.TopicType
		var err error

		switch connType {
		case connections.Publish:
			domainID, channelID, _, topicType, err = parser.ParsePublishTopic(ctx, req.topic, true)
		case connections.Subscribe:
			domainID, channelID, _, topicType, err = parser.ParseSubscribeTopic(ctx, req.topic, true)
		}
		if err != nil {
			return authorizeRes{}, err
		}

		if topicType == messaging.HealthType {
			return authorizeRes{authorized: true}, nil
		}

		ar := &grpcChannelsV1.AuthzReq{
			Type:       uint32(connType),
			ClientId:   req.externalID,
			ClientType: policies.ClientType,
			ChannelId:  channelID,
			DomainId:   domainID,
		}
		res, err := channels.Authorize(ctx, ar)
		if err != nil {
			return authorizeRes{}, err
		}

		return authorizeRes{authorized: res.GetAuthorized()}, nil
	}
}
