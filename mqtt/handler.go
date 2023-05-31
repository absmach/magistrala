// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt/redis"
	"github.com/mainflux/mainflux/pkg/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mproxy/pkg/session"
)

var _ session.Handler = (*handler)(nil)

const protocol = "mqtt"

const (
	LogInfoSubscribed   = "subscribed with client_id %s to topics %s"
	LogInfoUnsubscribed = "unsubscribed client_id %s from topics %s"
	LogInfoConnected    = "connected with client_id %s"
	LogInfoDisconnected = "disconnected client_id %s and username %s"
	LogInfoPublished    = "published with client_id %s to the topic %s"
)

var (
	channelRegExp                   = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)
	ErrMalformedSubtopic            = errors.New("malformed subtopic")
	ErrClientNotInitialized         = errors.New("client is not initialized")
	ErrMalformedTopic               = errors.New("malformed topic")
	ErrMissingClientID              = errors.New("client_id not found")
	ErrMissingTopicPub              = errors.New("failed to publish due to missing topic")
	ErrMissingTopicSub              = errors.New("failed to subscribe due to missing topic")
	ErrAuthentication               = errors.New("failed to perform authentication over the entity")
	ErrFailedConnect                = errors.New("failed to connect")
	ErrFailedSubscribe              = errors.New("failed to subscribe")
	ErrFailedUnsubscribe            = errors.New("failed to unsubscribe")
	ErrFailedPublish                = errors.New("failed to publish")
	ErrFailedDisconnect             = errors.New("failed to disconnect")
	ErrFailedPublishDisconnectEvent = errors.New("failed to publish disconnect event")
	ErrFailedParseSubtopic          = errors.New("failed to parse subtopic")
	ErrFailedPublishConnectEvent    = errors.New("failed to publish connect event")
	ErrFailedPublishToMsgBroker     = errors.New("failed to publish to mainflux message broker")
)

// Event implements events.Event interface
type handler struct {
	publishers []messaging.Publisher
	auth       auth.Client
	logger     logger.Logger
	es         redis.EventStore
}

// NewHandler creates new Handler entity
func NewHandler(publishers []messaging.Publisher, es redis.EventStore,
	logger logger.Logger, auth auth.Client) session.Handler {
	return &handler{
		es:         es,
		logger:     logger,
		publishers: publishers,
		auth:       auth,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the MQTT broker
func (h *handler) AuthConnect(ctx context.Context) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return ErrClientNotInitialized
	}

	if s.ID == "" {
		return ErrMissingClientID
	}

	thid, err := h.auth.Identify(ctx, string(s.Password))
	if err != nil {
		return err
	}

	if thid != s.Username {
		return errors.ErrAuthentication
	}

	if err := h.es.Connect(s.Username); err != nil {
		h.logger.Error(errors.Wrap(ErrFailedPublishConnectEvent, err).Error())
	}

	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the MQTT broker
func (h *handler) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return ErrClientNotInitialized
	}
	if topic == nil {
		return ErrMissingTopicPub
	}

	return h.authAccess(ctx, s.Username, *topic)
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker
func (h *handler) AuthSubscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return ErrClientNotInitialized
	}
	if topics == nil || *topics == nil {
		return ErrMissingTopicSub
	}

	for _, v := range *topics {
		if err := h.authAccess(ctx, s.Username, v); err != nil {
			return err
		}

	}

	return nil
}

// Connect - after client successfully connected
func (h *handler) Connect(ctx context.Context) {
	s, ok := session.FromContext(ctx)
	if !ok {
		h.logger.Error(errors.Wrap(ErrFailedConnect, ErrClientNotInitialized).Error())
		return
	}
	h.logger.Info(fmt.Sprintf(LogInfoConnected, s.ID))
}

// Publish - after client successfully published
func (h *handler) Publish(ctx context.Context, topic *string, payload *[]byte) {
	s, ok := session.FromContext(ctx)
	if !ok {
		h.logger.Error(errors.Wrap(ErrFailedPublish, ErrClientNotInitialized).Error())
		return
	}
	h.logger.Info(fmt.Sprintf(LogInfoPublished, s.ID, *topic))
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>

	channelParts := channelRegExp.FindStringSubmatch(*topic)
	if len(channelParts) < 2 {
		h.logger.Error(errors.Wrap(ErrFailedPublish, ErrMalformedTopic).Error())
		return
	}

	chanID := channelParts[1]
	subtopic := channelParts[2]

	subtopic, err := parseSubtopic(subtopic)
	if err != nil {
		h.logger.Error(errors.Wrap(ErrFailedParseSubtopic, err).Error())
		return
	}

	msg := messaging.Message{
		Protocol:  protocol,
		Channel:   chanID,
		Subtopic:  subtopic,
		Publisher: s.Username,
		Payload:   *payload,
		Created:   time.Now().UnixNano(),
	}

	for _, pub := range h.publishers {
		if err := pub.Publish(ctx, msg.Channel, &msg); err != nil {
			h.logger.Error(errors.Wrap(ErrFailedPublishToMsgBroker, err).Error())
		}
	}
}

// Subscribe - after client successfully subscribed
func (h *handler) Subscribe(ctx context.Context, topics *[]string) {
	s, ok := session.FromContext(ctx)
	if !ok {
		h.logger.Error(errors.Wrap(ErrFailedSubscribe, ErrClientNotInitialized).Error())
		return
	}
	h.logger.Info(fmt.Sprintf(LogInfoSubscribed, s.ID, strings.Join(*topics, ",")))
}

// Unsubscribe - after client unsubscribed
func (h *handler) Unsubscribe(ctx context.Context, topics *[]string) {
	s, ok := session.FromContext(ctx)
	if !ok {
		h.logger.Error(errors.Wrap(ErrFailedUnsubscribe, ErrClientNotInitialized).Error())
		return
	}
	h.logger.Info(fmt.Sprintf(LogInfoUnsubscribed, s.ID, strings.Join(*topics, ",")))
}

// Disconnect - connection with broker or client lost
func (h *handler) Disconnect(ctx context.Context) {
	s, ok := session.FromContext(ctx)
	if !ok {
		h.logger.Error(errors.Wrap(ErrFailedDisconnect, ErrClientNotInitialized).Error())
		return
	}
	h.logger.Error(fmt.Sprintf(LogInfoDisconnected, s.ID, s.Username))
	if err := h.es.Disconnect(s.Username); err != nil {
		h.logger.Error(errors.Wrap(ErrFailedPublishDisconnectEvent, err).Error())
	}
}

func (h *handler) authAccess(ctx context.Context, username string, topic string) error {
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	if !channelRegExp.Match([]byte(topic)) {
		return ErrMalformedTopic
	}

	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 1 {
		return ErrMalformedTopic
	}

	chanID := channelParts[1]
	return h.auth.Authorize(ctx, chanID, username)
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", ErrMalformedSubtopic
	}
	subtopic = strings.Replace(subtopic, "/", ".", -1)

	elems := strings.Split(subtopic, ".")
	filteredElems := []string{}
	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", ErrMalformedSubtopic
		}

		filteredElems = append(filteredElems, elem)
	}

	subtopic = strings.Join(filteredElems, ".")
	return subtopic, nil
}
