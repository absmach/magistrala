// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package rabbitmq_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/rabbitmq"
	"github.com/stretchr/testify/assert"
)

var (
	streamTopic = "test-topic"
	eventsChan  = make(chan map[string]interface{})
	logger      = mglog.NewMock()
	errFailed   = errors.New("failed")
	numEvents   = 100
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
	_, err := rabbitmq.NewPublisher(context.Background(), "http://invaliurl.com", stream)
	assert.NotNilf(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err), err)

	publisher, err := rabbitmq.NewPublisher(context.Background(), rabbitmqURL, stream)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	_, err = rabbitmq.NewSubscriber("http://invaliurl.com", stream, consumer, logger)
	assert.NotNilf(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err), err)

	subcriber, err := rabbitmq.NewSubscriber(rabbitmqURL, stream, consumer, logger)
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
			receivedEvent := <-eventsChan

			val := int64(receivedEvent["occurred_at"].(float64))
			if assert.WithinRange(t, time.Unix(0, val), time.Now().Add(-time.Second), time.Now().Add(time.Second)) {
				delete(receivedEvent, "occurred_at")
				delete(tc.event, "occurred_at")
			}

			assert.Equal(t, tc.event["temperature"], receivedEvent["temperature"])
			assert.Equal(t, tc.event["humidity"], receivedEvent["humidity"])
			assert.Equal(t, tc.event["sensor_id"], receivedEvent["sensor_id"])
			assert.Equal(t, tc.event["status"], receivedEvent["status"])
			assert.Equal(t, tc.event["timestamp"], receivedEvent["timestamp"])
			assert.Equal(t, tc.event["operation"], receivedEvent["operation"])

		default:
			assert.ErrorContains(t, err, tc.err.Error(), fmt.Sprintf("%s - expected error: %s", tc.desc, tc.err))
		}
	}
}

func TestPubsub(t *testing.T) {
	subcases := []struct {
		desc         string
		stream       string
		consumer     string
		errorMessage error
		handler      events.EventHandler
	}{
		{
			desc:         "Subscribe to a stream",
			stream:       fmt.Sprintf("%s.%s", stream, streamTopic),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to the same stream",
			stream:       fmt.Sprintf("%s.%s", stream, streamTopic),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an empty stream with an empty consumer",
			stream:       "",
			consumer:     "",
			errorMessage: rabbitmq.ErrEmptyStream,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to an empty stream with a valid consumer",
			stream:       "",
			consumer:     consumer,
			errorMessage: rabbitmq.ErrEmptyStream,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to a valid stream with an empty consumer",
			stream:       fmt.Sprintf("%s.%s", stream, streamTopic),
			consumer:     "",
			errorMessage: rabbitmq.ErrEmptyConsumer,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to another stream",
			stream:       fmt.Sprintf("%s.%s", stream, streamTopic+"1"),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{false},
		},
		{
			desc:         "Subscribe to a stream with malformed handler",
			stream:       fmt.Sprintf("%s.%s", stream, streamTopic),
			consumer:     consumer,
			errorMessage: nil,
			handler:      handler{true},
		},
	}

	for _, pc := range subcases {
		subcriber, err := rabbitmq.NewSubscriber(rabbitmqURL, pc.stream, pc.consumer, logger)
		if err != nil {
			assert.Equal(t, err, pc.errorMessage, fmt.Sprintf("%s got expected error: %s - got: %s", pc.desc, pc.errorMessage, err))

			continue
		}

		switch err := subcriber.Subscribe(context.Background(), pc.handler); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", pc.desc, err))
		default:
			assert.Equal(t, err, pc.errorMessage, fmt.Sprintf("%s got expected error: %s - got: %s", pc.desc, pc.errorMessage, err))
		}

		err = subcriber.Close()
		assert.Nil(t, err, fmt.Sprintf("%s got unexpected error: %s", pc.desc, err))
	}
}

func TestUnavailablePublish(t *testing.T) {
	publisher, err := rabbitmq.NewPublisher(context.Background(), rabbitmqURL, stream)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	subcriber, err := rabbitmq.NewSubscriber(rabbitmqURL, stream, consumer, logger)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on creating event store: %s", err))

	err = subcriber.Subscribe(context.Background(), handler{})
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on subscribing to event store: %s", err))

	err = pool.Client.PauseContainer(container.Container.ID)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on pausing container: %s", err))

	spawnGoroutines(publisher, t)

	time.Sleep(1 * time.Second)

	err = pool.Client.UnpauseContainer(container.Container.ID)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on unpausing container: %s", err))

	// Wait for the events to be published.
	time.Sleep(1 * time.Second)

	err = publisher.Close()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on closing publisher: %s", err))

	// read all the events from the channel and assert that they are 10.
	var receivedEvents []map[string]interface{}
	for i := 0; i < numEvents; i++ {
		event := <-eventsChan
		receivedEvents = append(receivedEvents, event)
	}
	assert.Len(t, receivedEvents, numEvents, "got unexpected number of events")
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
	for i := 0; i < numEvents; i++ {
		go func() {
			err := publisher.Publish(context.Background(), generateRandomEvent())
			assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		}()
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
