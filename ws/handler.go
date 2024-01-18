// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/mproxy/pkg/session"
)

var _ session.Handler = (*handler)(nil)

const protocol = "websocket"

// Log message formats.
const (
	LogInfoSubscribed   = "subscribed with client_id %s to topics %s"
	LogInfoUnsubscribed = "unsubscribed client_id %s from topics %s"
	LogInfoConnected    = "connected with client_id %s"
	LogInfoDisconnected = "disconnected client_id %s and username %s"
	LogInfoPublished    = "published with client_id %s to the topic %s"
)

// Error wrappers for MQTT errors.
var (
	ErrMalformedSubtopic            = errors.New("malformed subtopic")
	ErrClientNotInitialized         = errors.New("client is not initialized")
	ErrMalformedTopic               = errors.New("malformed topic")
	ErrMissingClientID              = errors.New("client_id not found")
	ErrMissingTopicPub              = errors.New("failed to publish due to missing topic")
	ErrMissingTopicSub              = errors.New("failed to subscribe due to missing topic")
	ErrFailedConnect                = errors.New("failed to connect")
	ErrFailedSubscribe              = errors.New("failed to subscribe")
	ErrFailedPublish                = errors.New("failed to publish")
	ErrFailedDisconnect             = errors.New("failed to disconnect")
	ErrFailedPublishDisconnectEvent = errors.New("failed to publish disconnect event")
	ErrFailedParseSubtopic          = errors.New("failed to parse subtopic")
	ErrFailedPublishConnectEvent    = errors.New("failed to publish connect event")
	ErrFailedPublishToMsgBroker     = errors.New("failed to publish to magistrala message broker")
)

var channelRegExp = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)

// Event implements events.Event interface.
type handler struct {
	pubsub messaging.PubSub
	auth   magistrala.AuthzServiceClient
	logger *slog.Logger
}

// NewHandler creates new Handler entity.
func NewHandler(pubsub messaging.PubSub, logger *slog.Logger, authClient magistrala.AuthzServiceClient) session.Handler {
	return &handler{
		logger: logger,
		pubsub: pubsub,
		auth:   authClient,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the ws server.
func (h *handler) AuthConnect(ctx context.Context) error {
	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the ws server.
func (h *handler) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	if topic == nil {
		return ErrMissingTopicPub
	}
	s, ok := session.FromContext(ctx)
	if !ok {
		return ErrClientNotInitialized
	}

	var token string
	switch {
	case strings.HasPrefix(string(s.Password), "Thing"):
		token = strings.ReplaceAll(string(s.Password), "Thing ", "")
	default:
		token = string(s.Password)
	}

	return h.authAccess(ctx, token, *topic, auth.PublishPermission)
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker.
func (h *handler) AuthSubscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return ErrClientNotInitialized
	}
	if topics == nil || *topics == nil {
		return ErrMissingTopicSub
	}

	var token string
	switch {
	case strings.HasPrefix(string(s.Password), "Thing"):
		token = strings.ReplaceAll(string(s.Password), "Thing ", "")
	default:
		token = string(s.Password)
	}

	for _, v := range *topics {
		if err := h.authAccess(ctx, token, v, auth.SubscribePermission); err != nil {
			return err
		}
	}

	return nil
}

// Connect - after client successfully connected.
func (h *handler) Connect(ctx context.Context) error {
	return nil
}

// Publish - after client successfully published.
func (h *handler) Publish(ctx context.Context, topic *string, payload *[]byte) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(ErrFailedPublish, ErrClientNotInitialized)
	}
	h.logger.Info(fmt.Sprintf(LogInfoPublished, s.ID, *topic))

	if len(*payload) == 0 {
		return ErrFailedMessagePublish
	}

	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	channelParts := channelRegExp.FindStringSubmatch(*topic)
	if len(channelParts) < 2 {
		return errors.Wrap(ErrFailedPublish, ErrMalformedTopic)
	}

	chanID := channelParts[1]
	subtopic := channelParts[2]

	subtopic, err := parseSubtopic(subtopic)
	if err != nil {
		return errors.Wrap(ErrFailedParseSubtopic, err)
	}

	var token string
	switch {
	case strings.HasPrefix(string(s.Password), "Thing"):
		token = strings.ReplaceAll(string(s.Password), "Thing ", "")
	default:
		token = string(s.Password)
	}

	ar := &magistrala.AuthorizeReq{
		SubjectType: auth.ThingType,
		Permission:  auth.PublishPermission,
		Subject:     token,
		Object:      chanID,
		ObjectType:  auth.GroupType,
	}
	res, err := h.auth.Authorize(ctx, ar)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}

	msg := messaging.Message{
		Protocol:  protocol,
		Channel:   chanID,
		Subtopic:  subtopic,
		Publisher: res.GetId(),
		Payload:   *payload,
		Created:   time.Now().UnixNano(),
	}

	if err := h.pubsub.Publish(ctx, msg.GetChannel(), &msg); err != nil {
		return errors.Wrap(ErrFailedPublishToMsgBroker, err)
	}

	return nil
}

// Subscribe - after client successfully subscribed.
func (h *handler) Subscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(ErrFailedSubscribe, ErrClientNotInitialized)
	}
	h.logger.Info(fmt.Sprintf(LogInfoSubscribed, s.ID, strings.Join(*topics, ",")))
	return nil
}

// Unsubscribe - after client unsubscribed.
func (h *handler) Unsubscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(ErrFailedUnsubscribe, ErrClientNotInitialized)
	}

	h.logger.Info(fmt.Sprintf(LogInfoUnsubscribed, s.ID, strings.Join(*topics, ",")))
	return nil
}

// Disconnect - connection with broker or client lost.
func (h *handler) Disconnect(ctx context.Context) error {
	return nil
}

func (h *handler) authAccess(ctx context.Context, password, topic, action string) error {
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	if !channelRegExp.MatchString(topic) {
		return ErrMalformedTopic
	}

	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 1 {
		return ErrMalformedTopic
	}

	chanID := channelParts[1]

	ar := &magistrala.AuthorizeReq{
		SubjectType: auth.ThingType,
		Permission:  action,
		Subject:     password,
		Object:      chanID,
		ObjectType:  auth.GroupType,
	}
	res, err := h.auth.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return nil
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", ErrMalformedSubtopic
	}
	subtopic = strings.ReplaceAll(subtopic, "/", ".")

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
