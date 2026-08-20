// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"errors"
	"testing"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDelivery records which acknowledgment a handle invocation settled on,
// standing in for a real AMQP delivery.
type fakeDelivery struct {
	acked     bool
	nacked    bool
	rejected  bool
	ackErr    error
	nackErr   error
	rejectErr error
}

func (f *fakeDelivery) Ack() error {
	f.acked = true
	return f.ackErr
}

func (f *fakeDelivery) Nack() error {
	f.nacked = true
	return f.nackErr
}

func (f *fakeDelivery) Reject() error {
	f.rejected = true
	return f.rejectErr
}

type stubHandler struct {
	err error
}

func (s stubHandler) Handle(context.Context, events.Event) error {
	return s.err
}

func TestHandleAcksSuccessfulEvent(t *testing.T) {
	msg := &fakeDelivery{}
	sub := &subQueueStore{}

	err := sub.handle(context.Background(), stubHandler{}, []byte(`{"event":"entity.update"}`), false, msg)

	require.NoError(t, err)
	assert.True(t, msg.acked)
	assert.False(t, msg.nacked)
	assert.False(t, msg.rejected)
}

func TestHandleNacksOnceThenRejectsOnRedelivery(t *testing.T) {
	sub := &subQueueStore{}
	handler := stubHandler{err: errors.New("invalidator failed")}
	body := []byte(`{"event":"entity.update","tenant_id":"domain-1"}`)

	first := &fakeDelivery{}
	err := sub.handle(context.Background(), handler, body, false, first)

	require.ErrorIs(t, err, handler.err)
	assert.False(t, first.acked)
	assert.True(t, first.nacked, "a first handler failure must Nack (requeue for one retry)")
	assert.False(t, first.rejected)

	second := &fakeDelivery{}
	err = sub.handle(context.Background(), handler, body, true, second)

	require.ErrorIs(t, err, handler.err)
	assert.False(t, second.acked)
	assert.False(t, second.nacked, "a redelivery must not be Nacked again")
	assert.True(t, second.rejected, "a persistent handler failure must be Rejected, not requeued forever")
}

func TestHandleRejectsUnparsableBody(t *testing.T) {
	msg := &fakeDelivery{}
	sub := &subQueueStore{}

	err := sub.handle(context.Background(), stubHandler{}, []byte("not json"), false, msg)

	require.Error(t, err)
	assert.False(t, msg.acked)
	assert.False(t, msg.nacked, "an unparsable body cannot succeed on retry and must not be requeued")
	assert.True(t, msg.rejected)
}

func TestHandlePropagatesAcknowledgmentErrors(t *testing.T) {
	sub := &subQueueStore{}
	handler := stubHandler{err: errors.New("invalidator failed")}
	body := []byte(`{"event":"entity.update","tenant_id":"domain-1"}`)

	msg := &fakeDelivery{nackErr: errors.New("channel closed")}
	err := sub.handle(context.Background(), handler, body, false, msg)

	require.ErrorIs(t, err, handler.err)
	require.ErrorIs(t, err, msg.nackErr)
}
