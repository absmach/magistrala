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
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt/redis"
	"github.com/mainflux/mproxy/pkg/events"
	opentracing "github.com/opentracing/opentracing-go"
)

var _ events.Event = (*Event)(nil)

var (
	channelRegExp         = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)
	ctRegExp              = regexp.MustCompile(`^(\/.*)?\/ct\/([^\/]+)$`)
	errMalformedTopic     = errors.New("malformed topic")
	errMalformedData      = errors.New("malformed request data")
	errMalformedSubtopic  = errors.New("malformed subtopic")
	errUnauthorizedAccess = errors.New("missing or invalid credentials provided")
	errInvalidConnect     = errors.New("CONENCT request with invalid username or client ID")
	errNilTopicPub        = errors.New("PUBLISH to nil topic")
	errNilTopicSub        = errors.New("SUB to nil topic")
)

// Event implements events.Event interface
type Event struct {
	tc     mainflux.ThingsServiceClient
	pubs   []mainflux.MessagePublisher
	tracer opentracing.Tracer
	logger logger.Logger
	es     redis.EventStore
}

// New creates new Event entity
func New(tc mainflux.ThingsServiceClient, pubs []mainflux.MessagePublisher, es redis.EventStore,
	logger logger.Logger, tracer opentracing.Tracer) *Event {
	return &Event{
		tc:     tc,
		pubs:   pubs,
		es:     es,
		tracer: tracer,
		logger: logger,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the MQTT broker
func (e *Event) AuthConnect(username, clientID *string, password *[]byte) error {
	if username == nil || clientID == nil {
		return errInvalidConnect
	}
	e.logger.Info(fmt.Sprintf("AuthConenct - client ID: %s, username: %s",
		*clientID, *username))

	t := &mainflux.Token{
		Value: string(*password),
	}

	thid, err := e.tc.Identify(context.TODO(), t)
	if err != nil {
		return err
	}

	if thid.Value != *username {
		return errUnauthorizedAccess
	}

	if err := e.es.Connect(*clientID); err != nil {
		e.logger.Warn("Failed to publish connect event: " + err.Error())
	}

	return nil
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

// AuthPublish is called on device publish,
// prior forwarding to the MQTT broker
func (e *Event) AuthPublish(username, clientID string, topic *string, payload *[]byte) error {
	if topic == nil {
		return errNilTopicPub
	}
	e.logger.Info("AuthPublish - client ID: " + clientID + " topic: " + *topic)
	return e.authAccess(username, *topic)
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker
func (e *Event) AuthSubscribe(username, clientID string, topics *[]string) error {
	if topics == nil || *topics == nil {
		return errNilTopicSub
	}
	e.logger.Info("AuthSubscribe - client ID: " + clientID + " topics: " + strings.Join(*topics, ","))

	for _, v := range *topics {
		if err := e.authAccess(username, v); err != nil {
			return err
		}

	}

	return nil
}

// Register - after client sucesfully connected
func (e *Event) Register(clientID string) {
	e.logger.Info("Register - client with ID: " + clientID)
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

// Publish - after client sucesfully published
func (e *Event) Publish(clientID, topic string, payload []byte) {
	e.logger.Info("Publish - client ID " + clientID + " to the topic: " + topic)
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>

	channelParts := channelRegExp.FindStringSubmatch(topic)
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

	msg := mainflux.Message{
		Protocol:    "mqtt",
		ContentType: ct,
		Channel:     chanID,
		Subtopic:    subtopic,
		Payload:     payload,
	}

	for _, mp := range e.pubs {
		go func(pub mainflux.MessagePublisher) {
			if err := pub.Publish(context.TODO(), "", msg); err != nil {
				e.logger.Info("Error publishing to Mainflux " + err.Error())
			}
		}(mp)
	}
}

// Subscribe - after client sucesfully subscribed
func (e *Event) Subscribe(clientID string, topics []string) {
	e.logger.Info("Subscribe - client ID: " + clientID + ", to topics: " + strings.Join(topics, ","))
}

// Unsubscribe - after client unsubscribed
func (e *Event) Unsubscribe(clientID string, topics []string) {
	e.logger.Info("Unubscribe - client ID: " + clientID + ", form topics: " + strings.Join(topics, ","))
}

// Disconnect - connection with broker or client lost
func (e *Event) Disconnect(clientID string) {
	if err := e.es.Disconnect(clientID); err != nil {
		e.logger.Warn("Failed to publish disconnect event: " + err.Error())
	}
}
