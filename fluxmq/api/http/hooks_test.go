// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/stretchr/testify/require"
)

type fakeHookParser struct {
	domainID        string
	channelID       string
	subtopic        string
	topicType       messaging.TopicType
	err             error
	publishCalled   bool
	subscribeCalled bool
}

func (p *fakeHookParser) ParsePublishTopic(context.Context, string, bool) (string, string, string, messaging.TopicType, error) {
	p.publishCalled = true
	return p.domainID, p.channelID, p.subtopic, p.topicType, p.err
}

func (p *fakeHookParser) ParseSubscribeTopic(context.Context, string, bool) (string, string, string, messaging.TopicType, error) {
	p.subscribeCalled = true
	return p.domainID, p.channelID, p.subtopic, p.topicType, p.err
}

func TestHooksHandlerReturnsCanonicalTopicModifier(t *testing.T) {
	parser := &fakeHookParser{
		domainID:  "26ad5c3f-cd91-4ff0-9685-0c3115643174",
		channelID: "cdc8f55f-0c54-4a9f-b4aa-8c69d4a8ce15",
		subtopic:  "messages",
		topicType: messaging.MessageType,
	}
	req := httptest.NewRequest("POST", "/hooks", strings.NewReader(`{
		"hook":"auth_on_publish",
		"client_id":"cli1",
		"external_id":"64d6bc95-b313-4412-9369-299543d9c63b",
		"protocol":"mqtt",
		"topic":"m/d1/c/ch1/messages"
	}`))
	w := httptest.NewRecorder()

	MakeHooksHandler(parser).ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.True(t, parser.publishCalled)
	require.False(t, parser.subscribeCalled)

	var res hookResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	require.Equal(t, hookResultOK, res.Result)
	require.Equal(t, "m/26ad5c3f-cd91-4ff0-9685-0c3115643174/c/cdc8f55f-0c54-4a9f-b4aa-8c69d4a8ce15/messages", res.Topic)
}

func TestHooksHandlerUsesSubscribeParserForSubscribeAndUnsubscribe(t *testing.T) {
	parser := &fakeHookParser{
		domainID:  "26ad5c3f-cd91-4ff0-9685-0c3115643174",
		channelID: "cdc8f55f-0c54-4a9f-b4aa-8c69d4a8ce15",
		subtopic:  "messages/+",
		topicType: messaging.MessageType,
	}
	for _, hook := range []string{hookAuthOnSubscribe, hookAuthOnUnsubscribe} {
		parser.publishCalled = false
		parser.subscribeCalled = false
		req := httptest.NewRequest("POST", "/hooks", strings.NewReader(`{"hook":"`+hook+`","topic":"m/d1/c/ch1/messages/+"}`))
		w := httptest.NewRecorder()

		MakeHooksHandler(parser).ServeHTTP(w, req)

		require.Equal(t, 200, w.Code)
		require.False(t, parser.publishCalled)
		require.True(t, parser.subscribeCalled)
	}
}

func TestHooksHandlerAllowsAMQP091MessageStreamWildcard(t *testing.T) {
	cases := []struct {
		desc     string
		protocol string
		topic    string
	}{
		{desc: "plain topic", protocol: "amqp091", topic: "m/#"},
		{desc: "leading slash topic", protocol: "amqp091", topic: "/m/#"},
		{desc: "uppercase protocol", protocol: "AMQP091", topic: "m/#"},
	}
	for _, tc := range cases {
		for _, hook := range []string{hookAuthOnSubscribe, hookAuthOnUnsubscribe} {
			parser := &fakeHookParser{err: errors.New("must not parse stream queue wildcard")}
			req := httptest.NewRequest("POST", "/hooks", strings.NewReader(`{"hook":"`+hook+`","protocol":"`+tc.protocol+`","topic":"`+tc.topic+`"}`))
			w := httptest.NewRecorder()

			MakeHooksHandler(parser).ServeHTTP(w, req)

			require.Equal(t, 200, w.Code, tc.desc)
			require.False(t, parser.publishCalled, tc.desc)
			require.False(t, parser.subscribeCalled, tc.desc)

			var res hookResponse
			require.NoError(t, json.NewDecoder(w.Body).Decode(&res), tc.desc)
			require.Equal(t, hookResultOK, res.Result, tc.desc)
			require.Equal(t, "m/#", res.Topic, tc.desc)
		}
	}
}

func TestHooksHandlerStillParsesMQTTMessageWildcard(t *testing.T) {
	parser := &fakeHookParser{err: errors.New("malformed topic")}
	req := httptest.NewRequest("POST", "/hooks", strings.NewReader(`{"hook":"auth_on_subscribe","protocol":"mqtt","topic":"m/#"}`))
	w := httptest.NewRecorder()

	MakeHooksHandler(parser).ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.False(t, parser.publishCalled)
	require.True(t, parser.subscribeCalled)

	var res hookResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	require.Equal(t, hookResultDeny, res.Result)
}

func TestHooksHandlerReturnsOKForNonMGTopic(t *testing.T) {
	req := httptest.NewRequest("POST", "/hooks", strings.NewReader(`{"hook":"auth_on_publish","topic":"$SYS/broker/uptime"}`))
	w := httptest.NewRecorder()

	MakeHooksHandler(nil).ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	var res hookResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	require.Equal(t, hookResultOK, res.Result)
	require.Empty(t, res.Topic)
}

func TestHooksHandlerDeniesUnresolvedMGTopic(t *testing.T) {
	parser := &fakeHookParser{err: errors.New("failed to resolve channel route")}
	req := httptest.NewRequest("POST", "/hooks", strings.NewReader(`{"hook":"auth_on_publish","topic":"m/d1/c/ch1/messages"}`))
	w := httptest.NewRecorder()

	MakeHooksHandler(parser).ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	var res hookResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	require.Equal(t, hookResultDeny, res.Result)
}
