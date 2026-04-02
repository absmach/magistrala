// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package messaging_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	chmocks "github.com/absmach/supermq/channels/mocks"
	dmocks "github.com/absmach/supermq/domains/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validRoute       = "valid-route"
	invalidRoute     = "invalid-route"
	channelID        = testsutil.GenerateUUID(&testing.T{})
	domainID         = testsutil.GenerateUUID(&testing.T{})
	topicFmt         = "m/%s/c/%s"
	healthTopicFmt   = "hc/%s"
	subtopic         = "subtopic"
	topicSubtopicFmt = "m/%s/c/%s/%s"
	cachedTopic      = fmt.Sprintf(topicSubtopicFmt, domainID, channelID, subtopic)
)

func setupResolver() (messaging.TopicResolver, *dmocks.DomainsServiceClient, *chmocks.ChannelsServiceClient) {
	channels := new(chmocks.ChannelsServiceClient)
	domains := new(dmocks.DomainsServiceClient)
	resolver := messaging.NewTopicResolver(channels, domains)

	return resolver, domains, channels
}

func setupParser() (messaging.TopicParser, *dmocks.DomainsServiceClient, *chmocks.ChannelsServiceClient, error) {
	channels := new(chmocks.ChannelsServiceClient)
	domains := new(dmocks.DomainsServiceClient)
	parser, err := messaging.NewTopicParser(messaging.DefaultCacheConfig, channels, domains)
	if err != nil {
		return nil, nil, nil, err
	}

	return parser, domains, channels, nil
}

var ParsePublisherTopicTestCases = []struct {
	desc      string
	topic     string
	domainID  string
	channelID string
	subtopic  string
	topicType messaging.TopicType
	err       error
}{
	{
		desc:      "valid topic with subtopic /m/domain123/c/channel456/devices/temp",
		topic:     "/m/domain123/c/channel456/devices/temp",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "devices/temp",
		topicType: messaging.MessageType,
		err:       nil,
	},
	{
		desc:      "valid topic with URL encoded subtopic /m/domain123/c/channel456/devices%2Ftemp%2Fdata",
		topic:     "/m/domain123/c/channel456/devices%2Ftemp%2Fdata",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "devices/temp/data",
		topicType: messaging.MessageType,
	},
	{
		desc:      "valid topic with subtopic /m/domain/c/channel/extra/extra2",
		topic:     "/m/domain/c/channel/extra/extra2",
		domainID:  "domain",
		channelID: "channel",
		subtopic:  "extra/extra2",
		topicType: messaging.MessageType,
	},
	{
		desc:      "valid topic without subtopic /m/domain123/c/channel456",
		topic:     "/m/domain123/c/channel456",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		topicType: messaging.MessageType,
	},
	{
		desc:      "valid topic with trailing slash /m/domain123/c/channel456/devices/temp/",
		topic:     "/m/domain123/c/channel456/devices/temp/",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "devices/temp",
		topicType: messaging.MessageType,
	},
	{
		desc:      "valid health check topic",
		topic:     fmt.Sprintf(healthTopicFmt, domainID),
		domainID:  domainID,
		channelID: "",
		subtopic:  "",
		topicType: messaging.HealthType,
		err:       nil,
	},
	{
		desc:      "invalid health check topic with empty domain",
		topic:     "hc/",
		domainID:  "",
		channelID: "",
		subtopic:  "",
		topicType: messaging.InvalidType,
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid topic format (missing parts) /m/domain123/c/",
		topic:     "/m/domain123/c/",
		domainID:  "domain123",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid topic format (missing domain) /m//c/channel123",
		topic:     "/m//c/channel123",
		domainID:  "",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid topic format (missing channel) /m/domain123/c/",
		topic:     "/m/domain123/c//subtopic",
		domainID:  "domain123",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "topic with wildcards + and # /m/domain123/c/channel456/devices/+/temp/#",
		topic:     "/m/domain123/c/channel456/devices/+/temp/#",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid domain name m/domain*123/c/channel456/devices/+/temp/#",
		topic:     "m/domain*123/c/channel456/devices/+/temp/#",
		domainID:  "",
		channelID: "channel456",
		subtopic:  "devices.*.temp.>",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a*b/topic",
		topic:     "/m/domain123/c/channel456/sub/a*b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a>b/topic",
		topic:     "/m/domain123/c/channel456/sub/a>b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a#b/topic",
		topic:     "/m/domain123/c/channel456/sub/a#b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a+b/topic",
		topic:     "/m/domain123/c/channel456/sub/a+b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a//b/topic",
		topic:     "/m/domain123/c/channel456/sub/a//b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid topic regex \"not-a-topic\"",
		topic:     "not-a-topic",
		domainID:  "",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:  "extra segment before prefix /extra/m/domain/c/channel",
		topic: "/extra/m/domain/c/channel",
		err:   messaging.ErrMalformedTopic,
	},
}

func TestParsePublishTopic(t *testing.T) {
	for _, tc := range ParsePublisherTopicTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			domainID, channelID, subtopic, topicType, err := messaging.ParsePublishTopic(tc.topic)
			assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			if err == nil {
				assert.Equal(t, tc.domainID, domainID)
				assert.Equal(t, tc.channelID, channelID)
				assert.Equal(t, tc.subtopic, subtopic)
				assert.Equal(t, tc.topicType, topicType)
			}
		})
	}
}

func BenchmarkParsePublisherTopic(b *testing.B) {
	for _, tc := range ParsePublisherTopicTestCases {
		b.Run(tc.desc, func(b *testing.B) {
			for b.Loop() {
				_, _, _, _, _ = messaging.ParsePublishTopic(tc.topic)
			}
		})
	}
}

var ParseSubscribeTestCases = []struct {
	desc      string
	topic     string
	domainID  string
	channelID string
	subtopic  string
	topicType messaging.TopicType
	err       error
}{
	{
		desc:      "valid topic with subtopic /m/domain123/c/channel456/devices/temp",
		topic:     "/m/domain123/c/channel456/devices/temp",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "devices/temp",
		topicType: messaging.MessageType,
	},
	{
		desc:      "topic with wildcards + and # /m/domain123/c/channel456/devices/+/temp/#",
		topic:     "/m/domain123/c/channel456/devices/+/temp/#",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "devices/+/temp/#",
		topicType: messaging.MessageType,
	},
	{
		desc:      "valid topic without subtopic /m/domain123/c/channel456",
		topic:     "/m/domain123/c/channel456",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		topicType: messaging.MessageType,
	},
	{
		desc:      "valid topic with trailing slash /m/domain123/c/channel456/devices/temp/",
		topic:     "/m/domain123/c/channel456/devices/temp/",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "devices/temp",
		topicType: messaging.MessageType,
	},
	{
		desc:      "valid health check topic",
		topic:     fmt.Sprintf(healthTopicFmt, domainID),
		domainID:  domainID,
		channelID: "",
		subtopic:  "",
		topicType: messaging.HealthType,
		err:       nil,
	},
	{
		desc:      "invalid health check topic with empty domain",
		topic:     "hc/",
		domainID:  "",
		channelID: "",
		subtopic:  "",
		topicType: messaging.InvalidType,
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid topic format (missing channel) /m/domain123/c/",
		topic:     "/m/domain123/c/",
		domainID:  "domain123",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid topic format (missing domain) /m//c/channel123",
		topic:     "/m//c/channel123",
		domainID:  "",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid topic format (missing channel) /m/domain123/c/",
		topic:     "/m/domain123/c//subtopic",
		domainID:  "domain123",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "valid domain with wildcards m/domain*123/c/channel456/devices/+/temp/#",
		topic:     "m/domain*123/c/channel456/devices/+/temp/#",
		domainID:  "domain*123",
		channelID: "channel456",
		subtopic:  "devices/+/temp/#",
		topicType: messaging.MessageType,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a*b/topic",
		topic:     "/m/domain123/c/channel456/sub/a*b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a>b/topic",
		topic:     "/m/domain123/c/channel456/sub/a>b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a#b/topic",
		topic:     "/m/domain123/c/channel456/sub/a#b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a+b/topic",
		topic:     "/m/domain123/c/channel456/sub/a+b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a//b/topic",
		topic:     "/m/domain123/c/channel456/sub/a//b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "invalid subtopic /m/domain123/c/channel456/sub/a/ /b/topic",
		topic:     "/m/domain123/c/channel456/sub/a/ /b/topic",
		domainID:  "domain123",
		channelID: "channel456",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:      "completely invalid topic \"invalid-topic\"",
		topic:     "invalid-topic",
		domainID:  "",
		channelID: "",
		subtopic:  "",
		err:       messaging.ErrMalformedTopic,
	},
	{
		desc:  "extra segment before prefix /extra/m/domain/c/channel",
		topic: "/extra/m/domain/c/channel",
		err:   messaging.ErrMalformedTopic,
	},
}

func TestParseSubscribeTopic(t *testing.T) {
	for _, tc := range ParseSubscribeTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			domainID, channelID, subtopic, topicType, err := messaging.ParseSubscribeTopic(tc.topic)
			assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			if err == nil {
				assert.Equal(t, tc.domainID, domainID)
				assert.Equal(t, tc.channelID, channelID)
				assert.Equal(t, tc.subtopic, subtopic)
				assert.Equal(t, tc.topicType, topicType)
			}
		})
	}
}

func BenchmarkParseSubscribeTopic(b *testing.B) {
	for _, tc := range ParseSubscribeTestCases {
		b.Run(tc.desc, func(b *testing.B) {
			for b.Loop() {
				_, _, _, _, _ = messaging.ParseSubscribeTopic(tc.topic)
			}
		})
	}
}

func TestEncodeTopic(t *testing.T) {
	cases := []struct {
		desc      string
		domainID  string
		channelID string
		subtopic  string
		expected  string
	}{
		{
			desc:      "with subtopic",
			domainID:  "domain1",
			channelID: "chan1",
			subtopic:  "dev/sensor/temp",
			expected:  "m/domain1/c/chan1/dev/sensor/temp",
		},
		{
			desc:      "without subtopic",
			domainID:  "domain1",
			channelID: "chan1",
			subtopic:  "",
			expected:  "m/domain1/c/chan1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := messaging.EncodeTopic(tc.domainID, tc.channelID, tc.subtopic)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestEncodeTopicSuffix(t *testing.T) {
	cases := []struct {
		desc      string
		domainID  string
		channelID string
		subtopic  string
		expected  string
	}{
		{
			desc:      "with subtopic",
			domainID:  "domain1",
			channelID: "chan1",
			subtopic:  "dev/sensor/temp",
			expected:  "domain1/c/chan1/dev/sensor/temp",
		},
		{
			desc:      "without subtopic",
			domainID:  "domain1",
			channelID: "chan1",
			subtopic:  "",
			expected:  "domain1/c/chan1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := messaging.EncodeTopicSuffix(tc.domainID, tc.channelID, tc.subtopic)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestMessage_EncodeTopicSuffix(t *testing.T) {
	cases := []struct {
		desc     string
		message  *messaging.Message
		expected string
	}{
		{
			desc: "with subtopic",
			message: &messaging.Message{
				Domain:   "domainX",
				Channel:  "chanX",
				Subtopic: "device/123/status",
			},
			expected: "domainX/c/chanX/device/123/status",
		},
		{
			desc: "without subtopic",
			message: &messaging.Message{
				Domain:  "domainY",
				Channel: "chanY",
			},
			expected: "domainY/c/chanY",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := messaging.EncodeMessageTopic(tc.message)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestMessage_EncodeToMQTTTopic(t *testing.T) {
	cases := []struct {
		desc     string
		message  *messaging.Message
		expected string
	}{
		{
			desc: "with subtopic",
			message: &messaging.Message{
				Domain:   "domainA",
				Channel:  "chanA",
				Subtopic: "dev/1/temp",
			},
			expected: "m/domainA/c/chanA/dev/1/temp",
		},
		{
			desc: "without subtopic",
			message: &messaging.Message{
				Domain:  "domainB",
				Channel: "chanB",
			},
			expected: "m/domainB/c/chanB",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := messaging.EncodeMessageMQTTTopic(tc.message)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestResolve(t *testing.T) {
	resolver, domains, channels := setupResolver()

	cases := []struct {
		desc        string
		domain      string
		channel     string
		domainID    string
		channelID   string
		isRoute     bool
		domainsErr  error
		channelsErr error
		err         error
	}{
		{
			desc:      "valid domainID and channelID",
			domain:    domainID,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			isRoute:   false,
			err:       nil,
		},
		{
			desc:      "valid domain route and channel ID",
			domain:    validRoute,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			isRoute:   true,
			err:       nil,
		},
		{
			desc:      "valid domain ID and channel route",
			domain:    domainID,
			channel:   validRoute,
			domainID:  domainID,
			channelID: channelID,
			isRoute:   true,
			err:       nil,
		},
		{
			desc:      "valid domain route and channel route",
			domain:    validRoute,
			channel:   validRoute,
			domainID:  domainID,
			channelID: channelID,
			isRoute:   true,
			err:       nil,
		},
		{
			desc:       "invalid domain route  and valid channel",
			domain:     invalidRoute,
			channel:    channelID,
			domainID:   "",
			channelID:  "",
			domainsErr: svcerr.ErrNotFound,
			err:        messaging.ErrFailedResolveDomain,
		},
		{
			desc:        "valid domain and invalid channel",
			domain:      domainID,
			channel:     invalidRoute,
			domainID:    domainID,
			channelID:   "",
			channelsErr: svcerr.ErrNotFound,
			err:         messaging.ErrFailedResolveChannel,
		},
		{
			desc:      "empty domain",
			domain:    "",
			channel:   channelID,
			domainID:  "",
			channelID: "",
			err:       messaging.ErrEmptyRouteID,
		},
		{
			desc:      "empty channel",
			domain:    domainID,
			channel:   "",
			domainID:  domainID,
			channelID: "",
			err:       nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			domainsCall := domains.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.domain}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.domainID,
				},
			}, tc.domainsErr)
			channelsCall := channels.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.channel, DomainId: tc.domainID}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.channelID,
				},
			}, tc.channelsErr)
			domainID, channelID, isRoute, err := resolver.Resolve(context.Background(), tc.domain, tc.channel)
			assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			if err == nil {
				assert.Equal(t, tc.domainID, domainID, "expected domain ID %s, got %s", tc.domainID, domainID)
				assert.Equal(t, tc.channelID, channelID, "expected channel ID %s, got %s", tc.channelID, channelID)
				assert.Equal(t, tc.isRoute, isRoute, "expected isRoute %t, got %t", tc.isRoute, isRoute)
			}
			domainsCall.Unset()
			channelsCall.Unset()
		})
	}
}

func TestResolveTopic(t *testing.T) {
	resolver, domains, channels := setupResolver()

	cases := []struct {
		desc        string
		topic       string
		domain      string
		channel     string
		domainID    string
		channelID   string
		domainsErr  error
		channelsErr error
		response    string
		err         error
	}{
		{
			desc:      "valid topic with domainID and channelID",
			topic:     fmt.Sprintf(topicFmt, domainID, channelID),
			domain:    domainID,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			response:  fmt.Sprintf(topicFmt, domainID, channelID),
			err:       nil,
		},
		{
			desc:      "valid topic with domain route and channel ID",
			topic:     fmt.Sprintf(topicFmt, validRoute, channelID),
			domain:    validRoute,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			response:  fmt.Sprintf(topicFmt, domainID, channelID),
			err:       nil,
		},
		{
			desc:      "valid topic with domain ID and channel route",
			topic:     fmt.Sprintf(topicFmt, domainID, validRoute),
			domain:    domainID,
			channel:   validRoute,
			domainID:  domainID,
			channelID: channelID,
			response:  fmt.Sprintf(topicFmt, domainID, channelID),
			err:       nil,
		},
		{
			desc:      "valid topic with domain route and channel route",
			topic:     fmt.Sprintf(topicFmt, validRoute, validRoute),
			domain:    validRoute,
			channel:   validRoute,
			domainID:  domainID,
			channelID: channelID,
			response:  fmt.Sprintf(topicFmt, domainID, channelID),
			err:       nil,
		},
		{
			desc:       "invalid topic with invalid domain route and valid channel",
			topic:      fmt.Sprintf(topicFmt, invalidRoute, channelID),
			domain:     invalidRoute,
			channel:    channelID,
			domainID:   "",
			channelID:  "",
			domainsErr: svcerr.ErrNotFound,
			err:        messaging.ErrFailedResolveDomain,
		},
		{
			desc:      "valid topic with valid topic with domainID and channelID and subtopic",
			topic:     fmt.Sprintf(topicFmt, domainID, channelID) + "/subtopic",
			domain:    domainID,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			response:  fmt.Sprintf(topicFmt, domainID, channelID) + "/subtopic",
			err:       nil,
		},
		{
			desc:      "invalid topic with empty domain",
			topic:     fmt.Sprintf(topicFmt, "", channelID),
			domain:    "",
			channel:   channelID,
			domainID:  "",
			channelID: "",
			err:       messaging.ErrMalformedTopic,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			domainsCall := domains.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.domain}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.domainID,
				},
			}, tc.domainsErr)
			channelsCall := channels.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.channel, DomainId: tc.domainID}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.channelID,
				},
			}, tc.channelsErr)
			rtopic, err := resolver.ResolveTopic(context.Background(), tc.topic)
			assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			if err == nil {
				assert.Equal(t, tc.response, rtopic, "expected topic %s, got %s", tc.response, rtopic)
			}
			domainsCall.Unset()
			channelsCall.Unset()
		})
	}
}

func TestParserPublishTopic(t *testing.T) {
	parser, domains, channels, err := setupParser()
	assert.Nil(t, err, fmt.Sprintf("unexpected error while setting up parser: %v", err))

	udomainID := testsutil.GenerateUUID(t)
	uchannelID := testsutil.GenerateUUID(t)

	cachedInvalidTopic := "m/invalid-domain/c"

	dom, ch, st, tt, err := parser.ParsePublishTopic(context.Background(), cachedTopic, false)
	assert.Nil(t, err, fmt.Sprintf("unexpected error while publishing topic: %v", err))
	assert.Equal(t, domainID, dom, "expected domainID %s, got %s", domainID, dom)
	assert.Equal(t, channelID, ch, "expected channelID %s, got %s", channelID, ch)
	assert.Equal(t, subtopic, st, "expected subtopic %s, got %s", subtopic, st)
	assert.Equal(t, messaging.MessageType, tt, "expected topic type %v, got %v", messaging.MessageType, tt)

	dom, ch, st, tt, err = parser.ParsePublishTopic(context.Background(), cachedInvalidTopic, false)
	assert.NotNil(t, err, "expected error for invalid cached topic")
	assert.Equal(t, "", dom, "expected empty domainID for invalid topic")
	assert.Equal(t, "", ch, "expected empty channelID for invalid topic")
	assert.Equal(t, "", st, "expected empty subtopic for invalid topic")
	assert.Equal(t, messaging.InvalidType, tt, "expected unknown topic type for invalid topic")
	time.Sleep(10 * time.Millisecond) // Ensure cache is populated

	cases := []struct {
		desc        string
		topic       string
		resolve     bool
		domain      string
		channel     string
		domainID    string
		channelID   string
		subtopic    string
		topicType   messaging.TopicType
		domainsErr  error
		channelsErr error
		err         error
	}{
		{
			desc:      "valid uncached topic with domainID and channelID",
			topic:     fmt.Sprintf(topicFmt, udomainID, uchannelID) + "/subtopic",
			resolve:   true,
			domain:    udomainID,
			channel:   uchannelID,
			domainID:  udomainID,
			channelID: uchannelID,
			subtopic:  subtopic,
			topicType: messaging.MessageType,
			err:       nil,
		},
		{
			desc:      "valid cached topic with domainID and channelID",
			topic:     cachedTopic,
			domain:    domainID,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			subtopic:  subtopic,
			topicType: messaging.MessageType,
			err:       nil,
		},
		{
			desc:      "invalid uncached topic with invalid format",
			topic:     "invalid-topic",
			domain:    "",
			channel:   "",
			domainID:  "",
			channelID: "",
			err:       messaging.ErrMalformedTopic,
		},
		{
			desc:      "invalid cached topic with invalid format",
			topic:     cachedInvalidTopic,
			domain:    "",
			channel:   "",
			domainID:  "",
			channelID: "",
			err:       messaging.ErrMalformedTopic,
		},
		{
			desc:      "valid uncached topic with domain and channel routes",
			topic:     fmt.Sprintf(topicFmt, validRoute, validRoute) + "/subtopic",
			resolve:   true,
			domain:    validRoute,
			channel:   validRoute,
			domainID:  domainID,
			channelID: channelID,
			subtopic:  subtopic,
			topicType: messaging.MessageType,
			err:       nil,
		},
		{
			desc:       "valid uncached topic with failed domain resolution",
			topic:      fmt.Sprintf(topicFmt, invalidRoute, uchannelID) + "/subtopic",
			resolve:    true,
			domain:     invalidRoute,
			channel:    uchannelID,
			domainID:   "",
			channelID:  "",
			domainsErr: svcerr.ErrNotFound,
			err:        messaging.ErrFailedResolveDomain,
		},
		{
			desc:      "valid uncached healthcheck topic",
			topic:     fmt.Sprintf(healthTopicFmt, domainID),
			domain:    domainID,
			channel:   "",
			domainID:  domainID,
			channelID: "",
			subtopic:  "",
			topicType: messaging.HealthType,
			err:       nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			domainsCall := domains.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.domain}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.domainID,
				},
			}, tc.domainsErr)
			channelsCall := channels.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.channel, DomainId: tc.domainID}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.channelID,
				},
			}, tc.channelsErr)
			domainID, channelID, subtopic, topicType, err := parser.ParsePublishTopic(context.Background(), tc.topic, tc.resolve)
			assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			if err == nil {
				assert.Equal(t, tc.domainID, domainID, "expected domainID %s, got %s", tc.domainID, domainID)
				assert.Equal(t, tc.channelID, channelID, "expected channelID %s, got %s", tc.channelID, channelID)
				assert.Equal(t, tc.subtopic, subtopic, "expected subtopic %s, got %s", tc.subtopic, subtopic)
				assert.Equal(t, tc.topicType, topicType, "expected topic type %v, got %v", tc.topicType, topicType)
			}
			domainsCall.Unset()
			channelsCall.Unset()
		})
	}
}

func BenchmarkParserPublishTopic(b *testing.B) {
	parser, _, _, err := setupParser()
	if err != nil {
		b.Fatalf("unexpected error while setting up parser: %v", err)
	}

	for _, tc := range ParsePublisherTopicTestCases {
		b.Run(tc.desc, func(b *testing.B) {
			for b.Loop() {
				_, _, _, _, _ = parser.ParsePublishTopic(context.Background(), tc.topic, false)
			}
		})
	}
}

func TestParserSubscribeTopic(t *testing.T) {
	parser, domains, channels, err := setupParser()
	assert.Nil(t, err, fmt.Sprintf("unexpected error while setting up parser: %v", err))

	cases := []struct {
		desc        string
		topic       string
		resolve     bool
		domain      string
		channel     string
		domainID    string
		channelID   string
		subtopic    string
		topicType   messaging.TopicType
		domainsErr  error
		channelsErr error
		err         error
	}{
		{
			desc:      "valid topic with domainID and channelID",
			topic:     fmt.Sprintf(topicFmt, domainID, channelID),
			resolve:   true,
			domain:    domainID,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			topicType: messaging.MessageType,
			err:       nil,
		},
		{
			desc:      "valid topic with domainID and channelID and subtopic",
			topic:     fmt.Sprintf(topicSubtopicFmt, domainID, channelID, subtopic),
			resolve:   true,
			domain:    domainID,
			channel:   channelID,
			domainID:  domainID,
			channelID: channelID,
			subtopic:  subtopic,
			topicType: messaging.MessageType,
			err:       nil,
		},
		{
			desc:      "valid topic with domain and channel routes",
			topic:     fmt.Sprintf(topicFmt, validRoute, validRoute),
			resolve:   true,
			domain:    validRoute,
			channel:   validRoute,
			domainID:  domainID,
			channelID: channelID,
			topicType: messaging.MessageType,
			err:       nil,
		},
		{
			desc:      "invalid topic with invalid format",
			topic:     "invalid-topic",
			resolve:   false,
			domain:    "",
			channel:   "",
			domainID:  "",
			channelID: "",
			err:       messaging.ErrMalformedTopic,
		},
		{
			desc:       "valid topic with invalid domain route",
			topic:      fmt.Sprintf(topicFmt, invalidRoute, validRoute),
			resolve:    true,
			domain:     invalidRoute,
			channel:    validRoute,
			domainID:   "",
			channelID:  "",
			domainsErr: svcerr.ErrNotFound,
			err:        messaging.ErrFailedResolveDomain,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			domainsCall := domains.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.domain}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.domainID,
				},
			}, tc.domainsErr)
			channelsCall := channels.On("RetrieveIDByRoute", mock.Anything, &grpcCommonV1.RetrieveIDByRouteReq{Route: tc.channel, DomainId: tc.domainID}).Return(&grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: tc.channelID,
				},
			}, tc.channelsErr)
			dom, ch, st, tt, err := parser.ParseSubscribeTopic(context.Background(), tc.topic, tc.resolve)
			assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			if err == nil {
				assert.Equal(t, tc.domainID, dom, "expected domainID %s, got %s", tc.domainID, dom)
				assert.Equal(t, tc.channelID, ch, "expected channelID %s, got %s", tc.channelID, ch)
				assert.Equal(t, tc.subtopic, st, "expected  subtopic %s, got %s", tc.subtopic, st)
				assert.Equal(t, tc.topicType, tt, "expected topic type %v, got %v", tc.topicType, tt)
			}
			domainsCall.Unset()
			channelsCall.Unset()
		})
	}
}
