// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
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

var (
	channelRegExp        = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)
	errMalformedTopic    = errors.New("malformed topic")
	errMalformedData     = errors.New("malformed request data")
	errMalformedSubtopic = errors.New("malformed subtopic")
	errNilClient         = errors.New("using nil client")
	errInvalidConnect    = errors.New("CONNECT request with invalid username or client ID")
	errNilTopicPub       = errors.New("PUBLISH to nil topic")
	errNilTopicSub       = errors.New("SUB to nil topic")
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
func (h *handler) AuthConnect(c *session.Client) error {
	if c == nil {
		return errInvalidConnect
	}

	thid, err := h.auth.Identify(context.Background(), string(c.Password))
	if err != nil {
		return err
	}

	if thid != c.Username {
		return errors.ErrAuthentication
	}

	if err := h.es.Connect(c.Username); err != nil {
		h.logger.Warn("Failed to publish connect event: " + err.Error())
	}

	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the MQTT broker
func (h *handler) AuthPublish(c *session.Client, topic *string, payload *[]byte) error {
	if c == nil {
		return errNilClient
	}
	if topic == nil {
		return errNilTopicPub
	}

	return h.authAccess(c.Username, *topic)
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker
func (h *handler) AuthSubscribe(c *session.Client, topics *[]string) error {
	if c == nil {
		return errNilClient
	}
	if topics == nil || *topics == nil {
		return errNilTopicSub
	}

	for _, v := range *topics {
		if err := h.authAccess(c.Username, v); err != nil {
			return err
		}

	}

	return nil
}

// Connect - after client successfully connected
func (h *handler) Connect(c *session.Client) {
	if c == nil {
		h.logger.Error("Nil client connect")
		return
	}
	h.logger.Info("Connect - client with ID: " + c.ID)
}

// Publish - after client successfully published
func (h *handler) Publish(c *session.Client, topic *string, payload *[]byte) {
	if c == nil {
		h.logger.Error("Nil client publish")
		return
	}
	h.logger.Info("Publish - client ID " + c.ID + " to the topic: " + *topic)
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>

	channelParts := channelRegExp.FindStringSubmatch(*topic)
	if len(channelParts) < 1 {
		h.logger.Info("Error in mqtt publish %s" + errMalformedData.Error())
		return
	}

	chanID := channelParts[1]
	subtopic := channelParts[2]

	subtopic, err := parseSubtopic(subtopic)
	if err != nil {
		h.logger.Info("Error parsing subtopic: " + err.Error())
		return
	}

	msg := messaging.Message{
		Protocol:  protocol,
		Channel:   chanID,
		Subtopic:  subtopic,
		Publisher: c.Username,
		Payload:   *payload,
		Created:   time.Now().UnixNano(),
	}

	for _, pub := range h.publishers {
		if err := pub.Publish(msg.Channel, msg); err != nil {
			h.logger.Info("Error publishing to Mainflux " + err.Error())
		}
	}
}

// Subscribe - after client successfully subscribed
func (h *handler) Subscribe(c *session.Client, topics *[]string) {
	if c == nil {
		h.logger.Error("Nil client subscribe")
		return
	}
	h.logger.Info("Subscribe - client ID: " + c.ID + ", to topics: " + strings.Join(*topics, ","))
}

// Unsubscribe - after client unsubscribed
func (h *handler) Unsubscribe(c *session.Client, topics *[]string) {
	if c == nil {
		h.logger.Error("Nil client unsubscribe")
		return
	}
	h.logger.Info("Unsubscribe - client ID: " + c.ID + ", form topics: " + strings.Join(*topics, ","))
}

// Disconnect - connection with broker or client lost
func (h *handler) Disconnect(c *session.Client) {
	if c == nil {
		h.logger.Error("Nil client disconnect")
		return
	}
	h.logger.Info("Disconnect - Client with ID: " + c.ID + " and username " + c.Username + " disconnected")
	if err := h.es.Disconnect(c.Username); err != nil {
		h.logger.Warn("Failed to publish disconnect event: " + err.Error())
	}
}

func (h *handler) authAccess(username string, topic string) error {
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	if !channelRegExp.Match([]byte(topic)) {
		h.logger.Info("Malformed topic: " + topic)
		return errMalformedTopic
	}

	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 1 {
		return errMalformedData
	}

	chanID := channelParts[1]
	return h.auth.Authorize(context.Background(), chanID, username)
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", errMalformedSubtopic
	}
	subtopic = strings.Replace(subtopic, "/", ".", -1)

	elems := strings.Split(subtopic, ".")
	filteredElems := []string{}
	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", errMalformedSubtopic
		}

		filteredElems = append(filteredElems, elem)
	}

	subtopic = strings.Join(filteredElems, ".")
	return subtopic, nil
}
