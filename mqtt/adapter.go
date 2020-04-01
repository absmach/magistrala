// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/broker"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt/redis"
	"github.com/mainflux/mproxy/pkg/session"
	opentracing "github.com/opentracing/opentracing-go"
)

var _ session.Event = (*Event)(nil)

var (
	channelRegExp         = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)
	ctRegExp              = regexp.MustCompile(`^(\/.*)?\/ct\/([^\/]+)$`)
	errMalformedTopic     = errors.New("malformed topic")
	errMalformedData      = errors.New("malformed request data")
	errMalformedSubtopic  = errors.New("malformed subtopic")
	errUnauthorizedAccess = errors.New("missing or invalid credentials provided")
	errNilClient          = errors.New("using nil client")
	errInvalidConnect     = errors.New("CONENCT request with invalid username or client ID")
	errNilTopicPub        = errors.New("PUBLISH to nil topic")
	errNilTopicSub        = errors.New("SUB to nil topic")
)

// Event implements events.Event interface
type Event struct {
	broker broker.Nats
	tc     mainflux.ThingsServiceClient
	tracer opentracing.Tracer
	logger logger.Logger
	es     redis.EventStore
}

// New creates new Event entity
func New(broker broker.Nats, tc mainflux.ThingsServiceClient, es redis.EventStore,
	logger logger.Logger, tracer opentracing.Tracer) *Event {
	return &Event{
		broker: broker,
		tc:     tc,
		es:     es,
		tracer: tracer,
		logger: logger,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the MQTT broker
func (e *Event) AuthConnect(c *session.Client) error {
	if c == nil {
		return errInvalidConnect
	}
	e.logger.Info(fmt.Sprintf("AuthConnect - client ID: %s, username: %s", c.ID, c.Username))

	t := &mainflux.Token{
		Value: string(c.Password),
	}

	thid, err := e.tc.Identify(context.TODO(), t)
	if err != nil {
		return err
	}

	if thid.Value != c.Username {
		return errUnauthorizedAccess
	}

	if err := e.es.Connect(c.Username); err != nil {
		e.logger.Warn("Failed to publish connect event: " + err.Error())
	}

	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the MQTT broker
func (e *Event) AuthPublish(c *session.Client, topic *string, payload *[]byte) error {
	if c == nil {
		return errNilClient
	}
	if topic == nil {
		return errNilTopicPub
	}
	e.logger.Info("AuthPublish - client ID: " + c.ID + " topic: " + *topic)
	return e.authAccess(c.Username, *topic)
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker
func (e *Event) AuthSubscribe(c *session.Client, topics *[]string) error {
	if c == nil {
		return errNilClient
	}
	if topics == nil || *topics == nil {
		return errNilTopicSub
	}
	e.logger.Info("AuthSubscribe - client ID: " + c.ID + " topics: " + strings.Join(*topics, ","))

	for _, v := range *topics {
		if err := e.authAccess(c.Username, v); err != nil {
			return err
		}

	}

	return nil
}

// Connect - after client sucesfully connected
func (e *Event) Connect(c *session.Client) {
	if c == nil {
		e.logger.Error("Nil client connect")
		return
	}
	e.logger.Info("Register - client with ID: " + c.ID)
}

// Publish - after client sucesfully published
func (e *Event) Publish(c *session.Client, topic *string, payload *[]byte) {
	if c == nil {
		e.logger.Error("Nil client publish")
		return
	}
	e.logger.Info("Publish - client ID " + c.ID + " to the topic: " + *topic)
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>

	channelParts := channelRegExp.FindStringSubmatch(*topic)
	if len(channelParts) < 1 {
		e.logger.Info("Error in mqtt publish %s" + errMalformedData.Error())
		return
	}

	chanID := channelParts[1]
	subtopic := channelParts[2]

	ct := ""
	if stParts := ctRegExp.FindStringSubmatch(subtopic); len(stParts) > 1 {
		ct = stParts[2]
		subtopic = stParts[1]
	}

	subtopic, err := parseSubtopic(subtopic)
	if err != nil {
		e.logger.Info("Error in mqtt publish: " + err.Error())
		return
	}

	msg := broker.Message{
		Protocol:    "mqtt",
		ContentType: ct,
		Channel:     chanID,
		Subtopic:    subtopic,
		Publisher:   c.Username,
		Payload:     *payload,
	}

	if err := e.broker.Publish(context.TODO(), "", msg); err != nil {
		e.logger.Info("Error publishing to Mainflux " + err.Error())
	}
}

// Subscribe - after client sucesfully subscribed
func (e *Event) Subscribe(c *session.Client, topics *[]string) {
	if c == nil {
		e.logger.Error("Nil client subscribe")
		return
	}
	e.logger.Info("Subscribe - client ID: " + c.ID + ", to topics: " + strings.Join(*topics, ","))
}

// Unsubscribe - after client unsubscribed
func (e *Event) Unsubscribe(c *session.Client, topics *[]string) {
	if c == nil {
		e.logger.Error("Nil client unsubscribe")
		return
	}
	e.logger.Info("Unubscribe - client ID: " + c.ID + ", form topics: " + strings.Join(*topics, ","))
}

// Disconnect - connection with broker or client lost
func (e *Event) Disconnect(c *session.Client) {
	if c == nil {
		e.logger.Error("Nil client disconnect")
		return
	}
	e.logger.Info("Disconnect - Client with ID: " + c.ID + " and username " + c.Username + " disconnected")
	if err := e.es.Disconnect(c.Username); err != nil {
		e.logger.Warn("Failed to publish disconnect event: " + err.Error())
	}
}

func (e *Event) authAccess(username string, topic string) error {
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	if !channelRegExp.Match([]byte(topic)) {
		e.logger.Info("Malformed topic: " + topic)
		return errMalformedTopic
	}

	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 1 {
		return errMalformedData
	}

	chanID := channelParts[1]

	ar := &mainflux.AccessByIDReq{
		ThingID: username,
		ChanID:  chanID,
	}
	_, err := e.tc.CanAccessByID(context.TODO(), ar)
	if err != nil {
		return err
	}

	return nil
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
