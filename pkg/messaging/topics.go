// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package messaging

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcCommonV1 "github.com/absmach/magistrala/api/grpc/common/v1"
	grpcDomainsV1 "github.com/absmach/magistrala/api/grpc/domains/v1"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/gofrs/uuid/v5"
)

const (
	MsgTopicPrefix     = 'm'
	ChannelTopicPrefix = 'c'
)

var (
	// MQTT wildcards are the canonical wildcard characters.
	mqttWildcards        = "+#"
	natsWildcards        = "*>"
	subtopicInvalidChars = " "
	subtopicSep          = "/"

	DefaultCacheConfig = CacheConfig{
		NumCounters: 2e5,     // 200k
		MaxCost:     1 << 20, // 1MB
		BufferItems: 64,
	}

	ErrMalformedTopic       = errors.New("malformed topic")
	ErrMalformedSubtopic    = errors.New("malformed subtopic")
	ErrEmptyRouteID         = errors.New("empty route or id")
	ErrFailedResolveDomain  = errors.New("failed to resolve domain route")
	ErrFailedResolveChannel = errors.New("failed to resolve channel route")
	ErrCreateCache          = errors.New("failed to create cache")
)

type TopicType uint8

const (
	InvalidType TopicType = iota
	MessageType
	HealthType
)

type CacheConfig struct {
	NumCounters int64 `env:"NUM_COUNTERS" envDefault:"200000"`  // number of keys to track frequency of.
	MaxCost     int64 `env:"MAX_COST"     envDefault:"1048576"` // maximum cost of cache.
	BufferItems int64 `env:"BUFFER_ITEMS" envDefault:"64"`      // number of keys per Get buffer.
}

type parsedTopic struct {
	domainID  string
	channelID string
	subtopic  string
	err       error
}

// TopicParser defines methods for parsing publish and subscribe topics.
// It uses a cache to store parsed topics for quick retrieval.
// It also resolves domain and channel IDs if requested.
type TopicParser interface {
	ParsePublishTopic(ctx context.Context, topic string, resolve bool) (domainID, channelID, subtopic string, topicType TopicType, err error)
	ParseSubscribeTopic(ctx context.Context, topic string, resolve bool) (domainID, channelID, subtopic string, topicType TopicType, err error)
}

type parser struct {
	resolver TopicResolver
	cache    *ristretto.Cache[string, *parsedTopic]
}

// NewTopicParser creates a new instance of TopicParser.
func NewTopicParser(cfg CacheConfig, channels grpcChannelsV1.ChannelsServiceClient, domains grpcDomainsV1.DomainsServiceClient) (TopicParser, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, *parsedTopic]{
		NumCounters: cfg.NumCounters,
		MaxCost:     cfg.MaxCost,
		BufferItems: cfg.BufferItems,
		Cost:        costFunc,
	})
	if err != nil {
		return nil, errors.Wrap(ErrCreateCache, err)
	}
	return &parser{
		cache:    cache,
		resolver: NewTopicResolver(channels, domains),
	}, nil
}

func (p *parser) ParsePublishTopic(ctx context.Context, topic string, resolve bool) (string, string, string, TopicType, error) {
	val, ok := p.cache.Get(topic)
	if ok {
		return val.domainID, val.channelID, val.subtopic, MessageType, val.err
	}
	domainID, channelID, subtopic, topicType, err := ParsePublishTopic(topic)
	if err != nil {
		p.saveToCache(topic, "", "", "", err)
		return "", "", "", InvalidType, err
	}
	var isRoute bool
	if resolve {
		domainID, channelID, isRoute, err = p.resolver.Resolve(ctx, domainID, channelID)
		if err != nil {
			return "", "", "", InvalidType, err
		}
	}
	if !isRoute && topicType == MessageType {
		p.saveToCache(topic, domainID, channelID, subtopic, nil)
	}

	return domainID, channelID, subtopic, topicType, nil
}

func (p *parser) ParseSubscribeTopic(ctx context.Context, topic string, resolve bool) (string, string, string, TopicType, error) {
	domainID, channelID, subtopic, topicType, err := ParseSubscribeTopic(topic)
	if err != nil {
		return "", "", "", InvalidType, err
	}
	if resolve {
		domainID, channelID, _, err = p.resolver.Resolve(ctx, domainID, channelID)
		if err != nil {
			return "", "", "", InvalidType, err
		}
	}

	return domainID, channelID, subtopic, topicType, nil
}

func (p *parser) saveToCache(topic string, domainID, channelID, subtopic string, err error) {
	p.cache.Set(topic, &parsedTopic{
		domainID:  domainID,
		channelID: channelID,
		subtopic:  subtopic,
		err:       err,
	}, 0)
}

func costFunc(val *parsedTopic) int64 {
	errLen := 0
	if val.err != nil {
		errLen = len(val.err.Error())
	}
	cost := int64(len(val.domainID) + len(val.channelID) + len(val.subtopic) + errLen)

	return cost
}

// TopicResolver contains definitions for resolving domain and channel IDs
// from their respective routes from the message topic.
type TopicResolver interface {
	Resolve(ctx context.Context, domain, channel string) (domainID string, channelID string, isRoute bool, err error)
	ResolveTopic(ctx context.Context, topic string) (rtopic string, err error)
}

type resolver struct {
	channels grpcChannelsV1.ChannelsServiceClient
	domains  grpcDomainsV1.DomainsServiceClient
}

// NewTopicResolver creates a new instance of TopicResolver.
func NewTopicResolver(channelsClient grpcChannelsV1.ChannelsServiceClient, domainsClient grpcDomainsV1.DomainsServiceClient) TopicResolver {
	return &resolver{
		channels: channelsClient,
		domains:  domainsClient,
	}
}

func (r *resolver) Resolve(ctx context.Context, domain, channel string) (string, string, bool, error) {
	if domain == "" {
		return "", "", false, ErrEmptyRouteID
	}

	domainID, isDomainRoute, err := r.resolveDomain(ctx, domain)
	if err != nil {
		return "", "", false, errors.Wrap(ErrFailedResolveDomain, err)
	}
	if channel == "" {
		return domainID, "", isDomainRoute, nil
	}
	channelID, isChannelRoute, err := r.resolveChannel(ctx, channel, domainID)
	if err != nil {
		return "", "", false, errors.Wrap(ErrFailedResolveChannel, err)
	}
	isRoute := isDomainRoute || isChannelRoute

	return domainID, channelID, isRoute, nil
}

func (r *resolver) ResolveTopic(ctx context.Context, topic string) (string, error) {
	domain, channel, subtopic, topicType, err := ParseTopic(topic)
	if err != nil {
		return "", errors.Wrap(ErrMalformedTopic, err)
	}

	domainID, channelID, _, err := r.Resolve(ctx, domain, channel)
	if err != nil {
		return "", err
	}
	rtopic := encodeAdapterTopic(domainID, channelID, subtopic, topicType)

	return rtopic, nil
}

func (r *resolver) resolveDomain(ctx context.Context, domain string) (string, bool, error) {
	if validateUUID(domain) == nil {
		return domain, false, nil
	}
	d, err := r.domains.RetrieveIDByRoute(ctx, &grpcCommonV1.RetrieveIDByRouteReq{
		Route: domain,
	})
	if err != nil {
		return "", false, err
	}

	return d.Entity.Id, true, nil
}

func (r *resolver) resolveChannel(ctx context.Context, channel, domainID string) (string, bool, error) {
	if validateUUID(channel) == nil {
		return channel, false, nil
	}
	c, err := r.channels.RetrieveIDByRoute(ctx, &grpcCommonV1.RetrieveIDByRouteReq{
		Route:    channel,
		DomainId: domainID,
	})
	if err != nil {
		return "", false, err
	}

	return c.Entity.Id, true, nil
}

func validateUUID(extID string) (err error) {
	id, err := uuid.FromString(extID)
	if id.String() != extID || err != nil {
		return err
	}

	return nil
}

func ParsePublishTopic(topic string) (domainID, chanID, subtopic string, topicType TopicType, err error) {
	domainID, chanID, subtopic, topicType, err = ParseTopic(topic)
	if err != nil {
		return "", "", "", InvalidType, err
	}
	subtopic, err = ParsePublishSubtopic(subtopic)
	if err != nil {
		return "", "", "", InvalidType, errors.Wrap(ErrMalformedTopic, err)
	}

	return domainID, chanID, subtopic, topicType, nil
}

func ParsePublishSubtopic(subtopic string) (parseSubTopic string, err error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err = formatSubtopic(subtopic)
	if err != nil {
		return "", errors.Wrap(ErrMalformedSubtopic, err)
	}

	if strings.ContainsAny(subtopic, subtopicInvalidChars+mqttWildcards+natsWildcards) {
		return "", ErrMalformedSubtopic
	}

	if strings.Contains(subtopic, "//") {
		return "", ErrMalformedSubtopic
	}

	return subtopic, nil
}

func ParseSubscribeTopic(topic string) (domainID string, chanID string, subtopic string, topicType TopicType, err error) {
	domainID, chanID, subtopic, topicType, err = ParseTopic(topic)
	if err != nil {
		return "", "", "", InvalidType, err
	}
	subtopic, err = ParseSubscribeSubtopic(subtopic)
	if err != nil {
		return "", "", "", InvalidType, errors.Wrap(ErrMalformedTopic, err)
	}

	return domainID, chanID, subtopic, topicType, nil
}

func ParseSubscribeSubtopic(subtopic string) (parseSubTopic string, err error) {
	if subtopic == "" {
		return "", nil
	}

	subtopic, err = formatSubtopic(subtopic)
	if err != nil {
		return "", errors.Wrap(ErrMalformedSubtopic, err)
	}

	if strings.ContainsAny(subtopic, subtopicInvalidChars+natsWildcards) {
		return "", ErrMalformedSubtopic
	}

	if strings.Contains(subtopic, "//") {
		return "", ErrMalformedSubtopic
	}

	for _, elem := range strings.Split(subtopic, subtopicSep) {
		if len(elem) > 1 && strings.ContainsAny(elem, mqttWildcards) {
			return "", ErrMalformedSubtopic
		}
	}
	return subtopic, nil
}

func formatSubtopic(subtopic string) (string, error) {
	subtopic, err := url.PathUnescape(subtopic)
	if err != nil {
		return "", err
	}
	subtopic = strings.TrimPrefix(subtopic, "/")
	subtopic = strings.TrimSuffix(subtopic, "/")
	subtopic = strings.TrimSpace(subtopic)
	return subtopic, nil
}

func EncodeTopic(domainID string, channelID string, subtopic string) string {
	return fmt.Sprintf("%s/%s", string(MsgTopicPrefix), EncodeTopicSuffix(domainID, channelID, subtopic))
}

func EncodeTopicSuffix(domainID string, channelID string, subtopic string) string {
	subject := fmt.Sprintf("%s/%s/%s", domainID, string(ChannelTopicPrefix), channelID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s/%s", subject, subtopic)
	}
	return subject
}

func EncodeMessageTopic(m *Message) string {
	return EncodeTopicSuffix(m.GetDomain(), m.GetChannel(), m.GetSubtopic())
}

func EncodeMessageMQTTTopic(m *Message) string {
	return EncodeTopic(m.GetDomain(), m.GetChannel(), m.GetSubtopic())
}

func encodeAdapterTopic(domain, channel, subtopic string, topicType TopicType) string {
	switch topicType {
	case HealthType:
		return fmt.Sprintf("%s/%s", string(HealthTopicPrefix), domain)
	default:
		topic := fmt.Sprintf("%s/%s/%s/%s", string(MsgTopicPrefix), domain, string(ChannelTopicPrefix), channel)
		if subtopic != "" {
			topic = topic + "/" + subtopic
		}
		return topic
	}
}

// ParseTopic parses a messaging topic string and returns the domain ID, channel ID, and subtopic.
// Supported formats (leading '/' optional):
//
//	m/<domain_id>/c/<channel_id>[/<subtopic>]
//	hc/<domain_id>
//
// This is an optimized version with no regex and minimal allocations.
func ParseTopic(topic string) (domainID, chanID, subtopic string, topicType TopicType, err error) {
	start := 0
	n := len(topic)
	if n > 0 && topic[0] == '/' {
		start = 1
	}
	if n <= start {
		return "", "", "", InvalidType, ErrMalformedTopic
	}

	// Healthcheck: "hc/<domain_id>"
	// Check first because it's shortest and avoids extra work.
	if n > start+3 && topic[start:start+2] == HealthTopicPrefix {
		if n == start+3 {
			// "hc/" with no domain
			return "", "", "", InvalidType, ErrMalformedTopic
		}
		// Domain is the remainder; ensure no extra '/'
		domainID = topic[start+3:]
		for i := start + 3; i < n; i++ {
			if topic[i] == '/' {
				return "", "", "", InvalidType, ErrMalformedTopic
			}
		}
		return domainID, "", "", HealthType, nil
	}

	// Messaging: "m/<domain_id>/c/<channel_id>[/<subtopic>]"
	// length check - minimum: "m/<domain_id>/c/" = 5 characters if ignore <domain_id> and in this case start will be 0
	// length check - minimum: "/m/<domain_id>/c/" = 6 characters if ignore <domain_id> and in this case start will be 1
	if n < start+5 {
		return "", "", "", InvalidType, ErrMalformedTopic
	}
	if topic[start] != MsgTopicPrefix || topic[start+1] != '/' {
		return "", "", "", InvalidType, ErrMalformedTopic
	}
	pos := start + 2

	// Find "/c/" to locate domain ID
	cPos := -1
	for i := pos; i <= n-3; i++ {
		if topic[i] == '/' && topic[i+1] == ChannelTopicPrefix && topic[i+2] == '/' {
			cPos = i - pos
			break
		}
	}
	if cPos == -1 || cPos == 0 {
		return "", "", "", InvalidType, ErrMalformedTopic
	}
	domainID = topic[pos : pos+cPos]
	// skip "/c/"
	pos = pos + cPos + 3

	// Ensure channel exists
	if pos >= n {
		return "", "", "", InvalidType, ErrMalformedTopic
	}

	// Find '/' after channelID
	nextSlash := -1
	for i := pos; i < n; i++ {
		if topic[i] == '/' {
			nextSlash = i - pos
			break
		}
	}

	if nextSlash == -1 {
		chanID = topic[pos:]
	} else {
		chanID = topic[pos : pos+nextSlash]
		subtopic = topic[pos+nextSlash+1:]
	}

	if len(chanID) == 0 {
		return "", "", "", InvalidType, ErrMalformedTopic
	}

	return domainID, chanID, subtopic, MessageType, nil
}
