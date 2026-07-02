// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	authv1 "github.com/absmach/fluxmq/pkg/proto/auth/v1"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/stretchr/testify/require"
)

type fakeTopicParser struct {
	domainID  string
	channelID string
	subtopic  string
	topicType messaging.TopicType
	err       error
}

func (p fakeTopicParser) ParsePublishTopic(context.Context, string, bool) (string, string, string, messaging.TopicType, error) {
	return p.domainID, p.channelID, p.subtopic, p.topicType, p.err
}

func (p fakeTopicParser) ParseSubscribeTopic(context.Context, string, bool) (string, string, string, messaging.TopicType, error) {
	return p.domainID, p.channelID, p.subtopic, p.topicType, p.err
}

type fakeAtomAuthorizer struct {
	resp atom.AuthzResponse
	err  error
}

func (a fakeAtomAuthorizer) CheckAuthz(context.Context, atom.AuthzRequest) (atom.AuthzResponse, error) {
	return a.resp, a.err
}

func TestAuthorizeReturnsAuthorizedOnlyWhenAllowed(t *testing.T) {
	srv := NewServer(nil, nil, fakeTopicParser{
		domainID:  "26ad5c3f-cd91-4ff0-9685-0c3115643174",
		channelID: "cdc8f55f-0c54-4a9f-b4aa-8c69d4a8ce15",
		subtopic:  "messages",
		topicType: messaging.MessageType,
	}, fakeAtomAuthorizer{resp: atom.AuthzResponse{Allowed: true}}).(*connectServer)

	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: "64d6bc95-b313-4412-9369-299543d9c63b",
		Topic:      "m/d1/c/ch1/messages",
		Action:     authv1.Action_Publish,
	}))
	require.NoError(t, err)
	require.True(t, res.Msg.GetAuthorized())
	require.Empty(t, res.Msg.ProtoReflect().GetUnknown())
}

func TestAuthorizeReturnsDeniedOnlyWhenDenied(t *testing.T) {
	srv := NewServer(nil, nil, fakeTopicParser{
		domainID:  "26ad5c3f-cd91-4ff0-9685-0c3115643174",
		channelID: "cdc8f55f-0c54-4a9f-b4aa-8c69d4a8ce15",
		subtopic:  "messages",
		topicType: messaging.MessageType,
	}, fakeAtomAuthorizer{resp: atom.AuthzResponse{Allowed: false}}).(*connectServer)

	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: "64d6bc95-b313-4412-9369-299543d9c63b",
		Topic:      "m/d1/c/ch1/messages",
		Action:     authv1.Action_Subscribe,
	}))
	require.NoError(t, err)
	require.False(t, res.Msg.GetAuthorized())
	require.Empty(t, res.Msg.ProtoReflect().GetUnknown())
}
