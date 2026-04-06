// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"testing"
	"time"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/magistrala/pkg/messaging"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

type testHandler struct {
	msg *messaging.Message
}

func (h *testHandler) Handle(msg *messaging.Message) error {
	h.msg = msg
	return nil
}

func (h *testHandler) Cancel() error {
	return nil
}

func TestHandleTopicMessageNormalizesAMQPRoutingKey(t *testing.T) {
	ps := &pubsub{
		publisher: publisher{
			options: options{prefix: "m"},
		},
	}
	h := &testHandler{}
	ts := time.Unix(1710000000, 123)

	err := ps.handleTopicMessage(h, &fluxamqp.Message{
		Delivery: amqp091.Delivery{
			Body:      []byte("payload"),
			Timestamp: ts,
			Headers: amqp091.Table{
				"external_id": "ext-user",
				"client_id":   "client-9",
			},
		},
		Topic: "m.domain.c.channel.test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.msg == nil {
		t.Fatal("expected handler to receive a message")
	}
	if h.msg.Domain != "domain" || h.msg.Channel != "channel" || h.msg.Subtopic != "test" {
		t.Fatalf("unexpected parsed message: %+v", h.msg)
	}
	if string(h.msg.Payload) != "payload" {
		t.Fatalf("unexpected payload: %q", string(h.msg.Payload))
	}
	if h.msg.Publisher != "ext-user" {
		t.Fatalf("unexpected publisher: %q", h.msg.Publisher)
	}
	if h.msg.GetClientId() != "client-9" {
		t.Fatalf("unexpected client ID: %q", h.msg.GetClientId())
	}
	if h.msg.Created != ts.UnixNano() {
		t.Fatalf("unexpected created timestamp: %d", h.msg.Created)
	}
}

func TestHandleTopicMessageUsesMQTTIdentityFields(t *testing.T) {
	ps := &pubsub{
		publisher: publisher{
			options: options{prefix: "m"},
		},
	}
	h := &testHandler{}
	ts := time.Unix(1710000000, 0)

	err := ps.handleTopicMessage(h, &fluxamqp.Message{
		Delivery: amqp091.Delivery{
			Body:      []byte("payload"),
			Timestamp: ts,
			Headers: amqp091.Table{
				"external_id": "ext-77",
				"client_id":   "client-7",
				"protocol":    "http",
				"created":     "1234567890000000000",
			},
		},
		Topic: "m.domain.c.channel.sub",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.msg.Publisher != "ext-77" {
		t.Fatalf("expected publisher from explicit header, got %q", h.msg.Publisher)
	}
	if h.msg.GetClientId() != "client-7" {
		t.Fatalf("expected client ID from header, got %q", h.msg.GetClientId())
	}
	if h.msg.Protocol != "http" {
		t.Fatalf("expected protocol from header, got %q", h.msg.Protocol)
	}
	if h.msg.Created != 1234567890000000000 {
		t.Fatalf("expected created from header, got %d", h.msg.Created)
	}
}

func TestMessageFromDelivery(t *testing.T) {
	cases := []struct {
		name      string
		body      []byte
		headers   map[string]any
		ts        time.Time
		prefix    string
		mqttTopic string
		want      *messaging.Message
		wantErr   bool
	}{
		{
			name:      "use explicit publisher and client_id headers",
			body:      []byte(`{"temperature":22.5}`),
			headers:   map[string]any{"external_id": "ext-1", "client_id": "client-1", "protocol": "mqtt", "created": "1710000000000000123"},
			ts:        time.Unix(1710000000, 0),
			prefix:    "writers",
			mqttTopic: "writers/domain/c/channel/temp",
			want: &messaging.Message{
				Domain:    "domain",
				Channel:   "channel",
				Subtopic:  "temp",
				Payload:   []byte(`{"temperature":22.5}`),
				Publisher: "ext-1",
				ClientId:  "client-1",
				Protocol:  "mqtt",
				Created:   1710000000000000123,
			},
		},
		{
			name:      "use explicit publisher header when present",
			body:      []byte("raw"),
			headers:   map[string]any{"external_id": "tenant-user", "client_id": "client-22"},
			ts:        time.Unix(1710000000, 250),
			prefix:    "m",
			mqttTopic: "m/dom/c/ch",
			want: &messaging.Message{
				Domain:    "dom",
				Channel:   "ch",
				Subtopic:  "",
				Payload:   []byte("raw"),
				Publisher: "tenant-user",
				ClientId:  "client-22",
				Protocol:  "mqtt",
				Created:   time.Unix(1710000000, 250).UnixNano(),
			},
		},
		{
			name:      "missing identity headers leaves publisher and client ID empty",
			body:      []byte("raw"),
			headers:   nil,
			ts:        time.Unix(1710000000, 500),
			prefix:    "m",
			mqttTopic: "m/dom/c/ch",
			want: &messaging.Message{
				Domain:    "dom",
				Channel:   "ch",
				Subtopic:  "",
				Payload:   []byte("raw"),
				Publisher: "",
				ClientId:  "",
				Protocol:  "mqtt",
				Created:   time.Unix(1710000000, 500).UnixNano(),
			},
		},
		{
			name:      "invalid topic",
			body:      []byte("x"),
			prefix:    "m",
			mqttTopic: "wrong/topic",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := messageFromDelivery(tc.body, tc.headers, tc.ts, tc.prefix, tc.mqttTopic)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Domain != tc.want.Domain || got.Channel != tc.want.Channel || got.Subtopic != tc.want.Subtopic {
				t.Fatalf("topic mismatch: got domain=%q channel=%q subtopic=%q", got.Domain, got.Channel, got.Subtopic)
			}
			if string(got.Payload) != string(tc.want.Payload) {
				t.Fatalf("payload mismatch: got %q, want %q", got.Payload, tc.want.Payload)
			}
			if got.Publisher != tc.want.Publisher {
				t.Fatalf("publisher mismatch: got %q, want %q", got.Publisher, tc.want.Publisher)
			}
			if got.GetClientId() != tc.want.GetClientId() {
				t.Fatalf("client ID mismatch: got %q, want %q", got.GetClientId(), tc.want.GetClientId())
			}
			if got.Protocol != tc.want.Protocol {
				t.Fatalf("protocol mismatch: got %q, want %q", got.Protocol, tc.want.Protocol)
			}
			if got.Created != tc.want.Created {
				t.Fatalf("created mismatch: got %d, want %d", got.Created, tc.want.Created)
			}
		})
	}
}
