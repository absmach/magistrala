// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

//go:build !nats && !rabbitmq
// +build !nats,!rabbitmq

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

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/redis"
	"github.com/stretchr/testify/assert"
)

var (
	streamName  = "magistrala.eventstest"
	consumer    = "test-consumer"
	streamTopic = "test-topic"
	eventsChan  = make(chan map[string]interface{})
	logger      = mglog.NewMock()
	errFailed   = errors.New("failed")
	ctx         = context.TODO()
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
	err := redisClient.FlushAll(ctx).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on flushing redis: %s", err))

	publisher, err := redis.NewPublisher(ctx, redisURL, streamName)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	subcriber, err := redis.NewSubscriber("http://invaliurl.com", streamName, consumer, logger)
	assert.NotNilf(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err), err)

	subcriber, err = redis.NewSubscriber(redisURL, streamName, consumer, logger)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	err = subcriber.Subscribe(ctx, handler{})
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

		err := publisher.Publish(ctx, event)
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

func TestUnavailablePublish(t *testing.T) {
	client, err := startContainer()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on starting container: %s", err))

	err = client.Client.FlushAll(ctx).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on flushing redis: %s", err))

	publisher, err := redis.NewPublisher(ctx, "http://invaliurl.com", streamName)
	assert.NotNilf(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err), err)

	publisher, err = redis.NewPublisher(ctx, client.url, streamName)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	err = client.pool.Client.PauseContainer(client.container.Container.ID)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on pausing container: %s", err))

	spawnGoroutines(publisher, t)

	err = client.pool.Client.UnpauseContainer(client.container.Container.ID)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on unpausing container: %s", err))

	// Wait for the events to be published.
	time.Sleep(events.UnpublishedEventsCheckInterval)

	err = publisher.Close()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on closing publisher: %s", err))

	err = client.pool.Purge(client.container)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on purging container: %s", err))
}

func generateRandomEvent() testEvent {
	return testEvent{
		Data: map[string]interface{}{
			"temperature": fmt.Sprintf("%f", rand.Float64()),
			"humidity":    fmt.Sprintf("%f", rand.Float64()),
			"sensor_id":   fmt.Sprintf("%d", rand.Intn(1000)),
			"location":    fmt.Sprintf("%f", rand.Float64()),
			"status":      fmt.Sprintf("%d", rand.Intn(1000)),
			"timestamp":   fmt.Sprintf("%d", time.Now().UnixNano()),
			"operation":   "create",
		},
	}
}

func spawnGoroutines(publisher events.Publisher, t *testing.T) {
	for i := 0; i < 1e4; i++ {
		go func() {
			for i := 0; i < 10; i++ {
				event := generateRandomEvent()
				err := publisher.Publish(ctx, event)
				assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
			}
		}()
	}
}

func TestPubsub(t *testing.T) {
	err := redisClient.FlushAll(ctx).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on flushing redis: %s", err))

	subcases := []struct {
		desc         string
		stream       string
		consumer     string
		errorMessage error
		handler      events.EventHandler
	}{
		{
			desc:         "Subscribe to a stream",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to the same stream",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an empty stream with an empty consumer",
			stream:       "",
			consumer:     "",
			errorMessage: redis.ErrEmptyStream,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an empty stream with a valid consumer",
			stream:       "",
			consumer:     consumer,
			errorMessage: redis.ErrEmptyStream,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to a valid stream with an empty consumer",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic),
			consumer:     "",
			errorMessage: redis.ErrEmptyConsumer,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to another stream",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic+"1"),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to a stream with malformed handler",
			stream:       fmt.Sprintf("%s.%s", streamName, streamTopic),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{true},
		},
	}

	for _, pc := range subcases {
		subcriber, err := redis.NewSubscriber(redisURL, pc.stream, pc.consumer, logger)
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

		err = subcriber.Close()
		assert.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", pc.desc, err))
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
