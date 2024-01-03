// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/mproxy/pkg/session"
)

var _ session.Handler = (*handler)(nil)

const protocol = "http"

// Log message formats.
const (
	LogInfoConnected = "connected with thing_key %s"
	// ThingPrefix represents the key prefix for Thing authentication scheme.
	ThingPrefix      = "Thing "
	LogInfoPublished = "published with client_id %s to the topic %s"
)

// Error wrappers for MQTT errors.
var (
	ErrMalformedSubtopic         = errors.New("malformed subtopic")
	ErrClientNotInitialized      = errors.New("client is not initialized")
	ErrMalformedTopic            = errors.New("malformed topic")
	ErrMissingTopicPub           = errors.New("failed to publish due to missing topic")
	ErrMissingTopicSub           = errors.New("failed to subscribe due to missing topic")
	ErrFailedConnect             = errors.New("failed to connect")
	ErrFailedPublish             = errors.New("failed to publish")
	ErrFailedParseSubtopic       = errors.New("failed to parse subtopic")
	ErrFailedPublishConnectEvent = errors.New("failed to publish connect event")
	ErrFailedPublishToMsgBroker  = errors.New("failed to publish to magistrala message broker")
)

var channelRegExp = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)

// Event implements events.Event interface.
type handler struct {
	publisher messaging.Publisher
	auth      magistrala.AuthzServiceClient
	logger    mglog.Logger
}

// NewHandler creates new Handler entity.
func NewHandler(publisher messaging.Publisher, logger mglog.Logger, authClient magistrala.AuthzServiceClient) session.Handler {
	return &handler{
		logger:    logger,
		publisher: publisher,
		auth:      authClient,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the HTTP server.
func (h *handler) AuthConnect(ctx context.Context) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return ErrClientNotInitialized
	}

	var tok string
	switch {
	case string(s.Password) == "":
		return errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerKey)
	case strings.HasPrefix(string(s.Password), "Thing"):
		tok = extractThingKey(string(s.Password))
	default:
		tok = string(s.Password)
	}

	h.logger.Info(fmt.Sprintf(LogInfoConnected, tok))
	return nil
}

// AuthPublish is not used in HTTP service.
func (h *handler) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	return nil
}

// AuthSubscribe is not used in HTTP service.
func (h *handler) AuthSubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Connect - after client successfully connected.
func (h *handler) Connect(ctx context.Context) error {
	return nil
}

// Publish - after client successfully published.
func (h *handler) Publish(ctx context.Context, topic *string, payload *[]byte) error {
	if topic == nil {
		return ErrMissingTopicPub
	}
	topic = &strings.Split(*topic, "?")[0]
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(ErrFailedPublish, ErrClientNotInitialized)
	}
	h.logger.Info(fmt.Sprintf(LogInfoPublished, s.ID, *topic))
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

	msg := messaging.Message{
		Protocol: protocol,
		Channel:  chanID,
		Subtopic: subtopic,
		Payload:  *payload,
		Created:  time.Now().UnixNano(),
	}
	var tok string
	switch {
	case string(s.Password) == "":
		return errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerKey)
	case strings.HasPrefix(string(s.Password), "Thing"):
		tok = extractThingKey(string(s.Password))
	default:
		tok = string(s.Password)
	}
	ar := &magistrala.AuthorizeReq{
		Subject:     tok,
		Object:      msg.Channel,
		SubjectType: auth.ThingType,
		Permission:  auth.PublishPermission,
		ObjectType:  auth.GroupType,
	}
	res, err := h.auth.Authorize(ctx, ar)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	msg.Publisher = res.GetId()

	if err := h.publisher.Publish(ctx, msg.Channel, &msg); err != nil {
		return errors.Wrap(ErrFailedPublishToMsgBroker, err)
	}

	return nil
}

// Subscribe - not used for HTTP.
func (h *handler) Subscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Unsubscribe - not used for HTTP.
func (h *handler) Unsubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Disconnect - not used for HTTP.
func (h *handler) Disconnect(ctx context.Context) error {
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

// extractThingKey returns value of the thing key. If there is no thing key - an empty value is returned.
func extractThingKey(topic string) string {
	if !strings.HasPrefix(topic, ThingPrefix) {
		return ""
	}

	return strings.TrimPrefix(topic, ThingPrefix)
}
