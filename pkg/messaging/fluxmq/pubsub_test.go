// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"reflect"
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
		Topic: "m.workspace.c.channel.test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.msg == nil {
		t.Fatal("expected handler to receive a message")
	}
	if h.msg.Workspace != "workspace" || h.msg.Channel != "channel" || h.msg.Subtopic != "test" {
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
		Topic: "m.workspace.c.channel.sub",
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
			name: "use explicit publisher and client_id headers",
			body: []byte(`{"temperature":22.5}`),
			headers: map[string]any{
				"external_id": "ext-1",
				"client_id":   "client-1",
				"protocol":    "mqtt",
				"created":     "1710000000000000123",
				headerMetadataPrefix + "magistrala.re.trace": `["rule-1"]`,
				headerMetadataPrefix + "invalid":             int64(1),
				"ordinary_header":                            "ignored",
			},
			ts:        time.Unix(1710000000, 0),
			prefix:    "writers",
			mqttTopic: "writers/workspace/c/channel/temp",
			want: &messaging.Message{
				Metadata:  map[string]string{"magistrala.re.trace": `["rule-1"]`},
				Workspace: "workspace",
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
			headers:   map[string]any{"external_id": "tenant-user", "client_id": "client-22", "created": int64(1710000000000000250)},
			ts:        time.Unix(1710000000, 250),
			prefix:    "m",
			mqttTopic: "m/dom/c/ch",
			want: &messaging.Message{
				Workspace: "dom",
				Channel:   "ch",
				Subtopic:  "",
				Payload:   []byte("raw"),
				Publisher: "tenant-user",
				ClientId:  "client-22",
				Protocol:  "mqtt",
				Created:   1710000000000000250,
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
				Workspace: "dom",
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
			if got.Workspace != tc.want.Workspace || got.Channel != tc.want.Channel || got.Subtopic != tc.want.Subtopic {
				t.Fatalf("topic mismatch: got workspace=%q channel=%q subtopic=%q", got.Workspace, got.Channel, got.Subtopic)
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
			if len(got.GetMetadata()) != len(tc.want.GetMetadata()) || got.GetMetadata()["magistrala.re.trace"] != tc.want.GetMetadata()["magistrala.re.trace"] {
				t.Fatalf("metadata mismatch: got %#v, want %#v", got.GetMetadata(), tc.want.GetMetadata())
			}
		})
	}
}

func TestMessagePropertiesIncludesMetadata(t *testing.T) {
	msg := &messaging.Message{
		Publisher: "publisher",
		Protocol:  "mqtt",
		ClientId:  "client",
		DeviceId:  "device",
		Created:   1710000000000000123,
		Metadata: map[string]string{
			"magistrala.re.trace": `["rule-1"]`,
		},
	}

	got := messageProperties(msg)

	want := map[string]string{
		headerExternalID: "publisher",
		headerProtocol:   "mqtt",
		"client_id":      "publisher",
		headerDeviceID:   "device",
		"created":        "1710000000000000123",
		headerMetadataPrefix + "magistrala.re.trace": `["rule-1"]`,
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("property %q mismatch: got %q, want %q", key, got[key], value)
		}
	}
	if len(got) != len(want) {
		t.Errorf("unexpected properties: %#v", got)
	}
}

func TestMetadataPropertyRoundTrip(t *testing.T) {
	want := &messaging.Message{
		Publisher: "rules-engine",
		Protocol:  "internal",
		ClientId:  "origin-client",
		DeviceId:  "origin-device",
		Created:   1710000000000000123,
		Payload:   []byte("payload"),
		Metadata: map[string]string{
			"magistrala.re.trace": "signed-trace",
			"other":               "preserved",
		},
	}

	properties := messageProperties(want)
	headers := make(map[string]any, len(properties))
	for key, value := range properties {
		headers[key] = value
	}

	got, err := messageFromDelivery(want.Payload, headers, time.Time{}, "m", "m/workspace/c/channel/subtopic")
	if err != nil {
		t.Fatalf("reconstruct message: %v", err)
	}
	if !reflect.DeepEqual(got.Metadata, want.Metadata) {
		t.Fatalf("metadata mismatch: got %#v, want %#v", got.Metadata, want.Metadata)
	}
	if got.DeviceId != want.DeviceId {
		t.Fatalf("device_id mismatch: got %q, want %q", got.DeviceId, want.DeviceId)
	}
}

func TestDeviceIDPropertyRoundTrip(t *testing.T) {
	want := &messaging.Message{
		Publisher: "gateway-1",
		Protocol:  "mqtt",
		Payload:   []byte("payload"),
		DeviceId:  "dev-1",
	}

	properties := messageProperties(want)
	headers := make(map[string]any, len(properties))
	for key, value := range properties {
		headers[key] = value
	}

	got, err := messageFromDelivery(want.Payload, headers, time.Time{}, "m", "m/workspace/c/channel/subtopic")
	if err != nil {
		t.Fatalf("reconstruct message: %v", err)
	}
	if got.GetDeviceId() != want.GetDeviceId() {
		t.Fatalf("device ID mismatch: got %q, want %q", got.GetDeviceId(), want.GetDeviceId())
	}
}

func TestMessagePropertiesOmitsDeviceIDWhenUnset(t *testing.T) {
	msg := &messaging.Message{Publisher: "gateway-1", Protocol: "mqtt"}

	got := messageProperties(msg)

	if _, ok := got[headerDeviceID]; ok {
		t.Fatalf("expected no %q property when DeviceId is unset, got %#v", headerDeviceID, got)
	}
}

func TestMessageFromDeliveryZeroTimestampFallsBackToNow(t *testing.T) {
	before := time.Now().UnixNano()
	got, err := messageFromDelivery([]byte("raw"), nil, time.Time{}, "m", "m/dom/c/ch")
	after := time.Now().UnixNano()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Created < before || got.Created > after {
		t.Fatalf("expected created timestamp between %d and %d, got %d", before, after, got.Created)
	}
}

func TestDirectTopicOnlyEnablesDirectIngressAndSkipsStream(t *testing.T) {
	ps := &pubsub{}
	if err := DirectTopicOnly()(ps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ps.directTopicIngress {
		t.Fatal("expected direct topic ingress to be enabled")
	}
	if !ps.directTopicOnly {
		t.Fatal("expected stream consumption to be skipped")
	}
}
