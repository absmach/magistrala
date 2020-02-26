// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mproxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt/mproxy/redis"
	"github.com/mainflux/mproxy/pkg/events"
	opentracing "github.com/opentracing/opentracing-go"
)

var (
	_                    events.Event = (*Event)(nil)
	channelRegExp                     = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)
	ctRegExp                          = regexp.MustCompile(`^(\/.*)?\/ct\/([^\/]+)$`)
	errMalformedTopic                 = errors.New("malformed topic")
	errMalformedData                  = errors.New("malformed request data")
	errMalformedSubtopic              = errors.New("malformed subtopic")
)

// Event implements events.Event interface
type Event struct {
	tc     mainflux.ThingsServiceClient
	mp     mainflux.MessagePublisher
	tracer opentracing.Tracer
	logger logger.Logger
	es     redis.EventStore
}

// New creates new Event entity
func New(tc mainflux.ThingsServiceClient, mp mainflux.MessagePublisher, es redis.EventStore,
	logger logger.Logger, tracer opentracing.Tracer) *Event {
	return &Event{
		tc:     tc,
		mp:     mp,
		es:     es,
		tracer: tracer,
		logger: logger,
	}
}

// AuthRegister is called on device connection,
// prior forwarding to the MQTT broker
func (e *Event) AuthRegister(username, clientID *string, password *[]byte) error {
	e.logger.Info(fmt.Sprintf("AuthRegister() - clientID: %s, username: %s",
		*clientID, *username))

	t := &mainflux.Token{
		Value: string(*password),
	}

	thid, err := e.tc.Identify(context.TODO(), t)
	if err != nil {
		return err
	}

	if thid.Value != *username {
		return err
	}

	return nil
}

func (e *Event) authAccess(username string, topic string) error {
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	if !channelRegExp.Match([]byte(topic)) {
		e.logger.Info(fmt.Sprintf("Malformed topic %s", topic))
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
	e.logger.Info(fmt.Sprintf("AuthPublish() - clientID: %s, topic: %s", clientID, *topic))
	return e.authAccess(username, *topic)
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker
func (e *Event) AuthSubscribe(username, clientID string, topics *[]string) error {
	e.logger.Info(fmt.Sprintf("AuthSubscribe() - clientID: %s, topics: %s", clientID, strings.Join(*topics, ",")))

	for _, v := range *topics {
		if err := e.authAccess(username, v); err != nil {
			return err
		}

	}

	return nil
}

// Register - after client sucesfully connected
func (e *Event) Register(clientID string) {
	e.logger.Info(fmt.Sprintf("Register() - clientID: %s", clientID))
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
	e.logger.Info(fmt.Sprintf("Publish() - clientID: %s, topic: %s", clientID, topic))
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>

	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 1 {
		e.logger.Info(fmt.Sprintf("Error in mqtt publish %s", errMalformedData))
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
		e.logger.Info(fmt.Sprintf("Error in mqtt publish %s", err))
		return
	}

	msg := mainflux.Message{
		Protocol:    "mqtt",
		ContentType: ct,
		Channel:     chanID,
		Subtopic:    subtopic,
		Payload:     payload,
	}

	if err := e.mp.Publish(context.TODO(), "", msg); err != nil {
		e.logger.Info(fmt.Sprintf("Error in mqtt publish %s", err))
		return
	}
}

// Subscribe - after client sucesfully subscribed
func (e *Event) Subscribe(clientID string, topics []string) {
	e.logger.Info(fmt.Sprintf("Subscribe() - clientID: %s, topics: %s", clientID, strings.Join(topics, ",")))
}

// Unubscribe - after client unsubscribed
func (e *Event) Unsubscribe(clientID string, topics []string) {

	e.logger.Info(fmt.Sprintf("Unubscribe() - clientID: %s, topics: %s", clientID, strings.Join(topics, ",")))
}
