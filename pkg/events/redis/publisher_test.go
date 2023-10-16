// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/events"
	"github.com/mainflux/mainflux/pkg/events/redis"
	"github.com/stretchr/testify/assert"
)

var (
	streamName  = "mainflux.eventstest"
	consumer    = "test-consumer"
	streamTopic = "test-topic"
	eventsChan  = make(chan map[string]interface{})
	logger      = mflog.NewMock()
	errFailed   = errors.New("failed")
)

type testEvent struct {
	Data map[string]interface{}
}

func (te testEvent) Encode() (map[string]interface{}, error) {
	data := make(map[string]interface{})
	for k, v := range te.Data {
		switch v.(type) {
		case string:
			data[k] = v
		case float64:
			data[k] = v
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			data[k] = string(b)
		}
	}

	return data, nil
}

func TestPublish(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on flushing redis: %s", err))

	publisher, err := redis.NewPublisher(context.Background(), redisURL, streamName)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	subcriber, err := redis.NewSubscriber(redisURL, streamName, consumer, logger)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	err = subcriber.Subscribe(context.Background(), handler{})
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on subscribing to event store: %s", err))

	cases := []struct {
		desc  string
		event map[string]interface{}
		err   error
	}{
		{
			desc: "publish event successfully",
			err:  nil,
			event: map[string]interface{}{
				"temperature": fmt.Sprintf("%f", rand.Float64()),
				"humidity":    fmt.Sprintf("%f", rand.Float64()),
				"sensor_id":   "abc123",
				"location":    "Earth",
				"status":      "normal",
				"timestamp":   fmt.Sprintf("%d", time.Now().UnixNano()),
				"operation":   "create",
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:  "publish with nil event",
			err:   nil,
			event: nil,
		},
		{
			desc: "publish event with invalid event location",
			err:  fmt.Errorf("json: unsupported type: chan int"),
			event: map[string]interface{}{
				"temperature": fmt.Sprintf("%f", rand.Float64()),
				"humidity":    fmt.Sprintf("%f", rand.Float64()),
				"sensor_id":   "abc123",
				"location":    make(chan int),
				"status":      "normal",
				"timestamp":   "invalid",
				"operation":   "create",
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc: "publish event with nested sting value",
			err:  nil,
			event: map[string]interface{}{
				"temperature": fmt.Sprintf("%f", rand.Float64()),
				"humidity":    fmt.Sprintf("%f", rand.Float64()),
				"sensor_id":   "abc123",
				"location": map[string]string{
					"lat": fmt.Sprintf("%f", rand.Float64()),
					"lng": fmt.Sprintf("%f", rand.Float64()),
				},
				"status":      "normal",
				"timestamp":   "invalid",
				"operation":   "create",
				"occurred_at": time.Now().UnixNano(),
			},
		},
	}

	for _, tc := range cases {
		event := testEvent{Data: tc.event}

		err := publisher.Publish(context.Background(), event)
		switch tc.err {
		case nil:
			assert.Nil(t, err, fmt.Sprintf("%s - got unexpected error: %s", tc.desc, err))

			receivedEvent := <-eventsChan

			roa, err := strconv.ParseInt(receivedEvent["occurred_at"].(string), 10, 64)
			assert.Nil(t, err, fmt.Sprintf("%s - got unexpected error: %s", tc.desc, err))
			if assert.WithinRange(t, time.Unix(0, roa), time.Now().Add(-time.Second), time.Now().Add(time.Second)) {
				delete(receivedEvent, "occurred_at")
				delete(tc.event, "occurred_at")
			}

			assert.Equal(t, tc.event["temperature"], receivedEvent["temperature"], fmt.Sprintf("%s - expected temperature: %s, got: %s", tc.desc, tc.event["temperature"], receivedEvent["temperature"]))
			assert.Equal(t, tc.event["humidity"], receivedEvent["humidity"], fmt.Sprintf("%s - expected humidity: %s, got: %s", tc.desc, tc.event["humidity"], receivedEvent["humidity"]))
			assert.Equal(t, tc.event["sensor_id"], receivedEvent["sensor_id"], fmt.Sprintf("%s - expected sensor_id: %s, got: %s", tc.desc, tc.event["sensor_id"], receivedEvent["sensor_id"]))
			assert.Equal(t, tc.event["status"], receivedEvent["status"], fmt.Sprintf("%s - expected status: %s, got: %s", tc.desc, tc.event["status"], receivedEvent["status"]))
			assert.Equal(t, tc.event["timestamp"], receivedEvent["timestamp"], fmt.Sprintf("%s - expected timestamp: %s, got: %s", tc.desc, tc.event["timestamp"], receivedEvent["timestamp"]))
			assert.Equal(t, tc.event["operation"], receivedEvent["operation"], fmt.Sprintf("%s - expected operation: %s, got: %s", tc.desc, tc.event["operation"], receivedEvent["operation"]))

		default:
			assert.ErrorContains(t, err, tc.err.Error(), fmt.Sprintf("%s - expected error: %s", tc.desc, tc.err))
		}
	}
}

func TestPubsub(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on flushing redis: %s", err))

	subcases := []struct {
		desc         string
		stream       string
		errorMessage error
		handler      events.EventHandler
	}{
		{
			desc:         "Subscribe to a stream",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic),
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to the same stream",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic),
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an empty stream",
			stream:       "",
			errorMessage: redis.ErrEmptyStream,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to another stream",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic+"1"),
			errorMessage: nil,
			handler:      handler{true},
		},
	}

	for _, pc := range subcases {
		subcriber, err := redis.NewSubscriber(redisURL, pc.stream, consumer, logger)
		if err != nil {
			assert.Equal(t, err, pc.errorMessage, fmt.Sprintf("%s got expected error: %s - got: %s", pc.desc, pc.errorMessage, err))

			continue
		}

		assert.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", pc.desc, err))

		switch err := subcriber.Subscribe(context.TODO(), pc.handler); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", pc.desc, err))
		default:
			assert.Equal(t, err, pc.errorMessage, fmt.Sprintf("%s got expected error: %s - got: %s", pc.desc, pc.errorMessage, err))
		}
	}
}

type handler struct {
	fail bool
}

func (h handler) Handle(_ context.Context, event events.Event) error {
	if h.fail {
		return errFailed
	}
	data, err := event.Encode()
	if err != nil {
		return err
	}

	eventsChan <- data

	return nil
}
