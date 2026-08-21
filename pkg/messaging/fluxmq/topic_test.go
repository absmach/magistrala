// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"strings"
	"testing"
)

func TestStreamFilter(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		topic  string
		want   string
	}{
		{
			name:   "all messages with prefix",
			prefix: "m",
			topic:  "m/#",
			want:   "#",
		},
		{
			name:   "all messages without explicit prefix",
			prefix: "writers",
			topic:  "#",
			want:   "#",
		},
		{
			name:   "specific topic filter",
			prefix: "writers",
			topic:  "writers/workspace/c/channel/+",
			want:   "workspace/c/channel/+",
		},
		{
			name:   "topic without prefix",
			prefix: "alarms",
			topic:  "workspace/c/channel/#",
			want:   "workspace/c/channel/#",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := streamFilter(tc.prefix, tc.topic)
			if got != tc.want {
				t.Fatalf("stream filter mismatch: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestQueueFilter(t *testing.T) {
	got := queueFilter("writers", "writers/#")
	want := "$queue/writers/#"
	if got != want {
		t.Fatalf("queue filter mismatch: got %q, want %q", got, want)
	}
}

func TestTopicFilter(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		topic  string
		want   string
	}{
		{
			name:   "all messages with prefix",
			prefix: "m",
			topic:  "m/#",
			want:   "m/#",
		},
		{
			name:   "wildcard topic",
			prefix: "writers",
			topic:  "#",
			want:   "writers/#",
		},
		{
			name:   "specific topic",
			prefix: "m",
			topic:  "m/workspace/c/channel/subtopic",
			want:   "m/workspace/c/channel/subtopic",
		},
		{
			name:   "single-level wildcard",
			prefix: "m",
			topic:  "m/workspace/c/+/subtopic",
			want:   "m/workspace/c/+/subtopic",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := topicFilter(tc.prefix, tc.topic)
			if got != tc.want {
				t.Fatalf("topic filter mismatch: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseMQTTTopic(t *testing.T) {
	cases := []struct {
		name      string
		prefix    string
		topic     string
		workspace string
		channel   string
		subtopic  string
		shouldErr bool
	}{
		{
			name:      "default prefix with subtopic path",
			prefix:    "m",
			topic:     "m/workspace/c/channel/sub/topic",
			workspace: "workspace",
			channel:   "channel",
			subtopic:  "sub/topic",
		},
		{
			name:      "alternate prefix without subtopic",
			prefix:    "writers",
			topic:     "writers/workspace/c/channel",
			workspace: "workspace",
			channel:   "channel",
			subtopic:  "",
		},
		{
			name:      "leading slash is ignored",
			prefix:    "alarms",
			topic:     "/alarms/workspace/c/channel/critical/high",
			workspace: "workspace",
			channel:   "channel",
			subtopic:  "critical/high",
		},
		{
			name:      "mismatched prefix",
			prefix:    "writers",
			topic:     "m/workspace/c/channel",
			shouldErr: true,
		},
		{
			name:      "invalid shape",
			prefix:    "m",
			topic:     "m/workspace/channel",
			shouldErr: true,
		},
		{
			name:      "empty subtopic segment",
			prefix:    "m",
			topic:     "m/workspace/c/channel/sub//topic",
			shouldErr: true,
		},
		{
			name:      "dot topic is invalid",
			prefix:    "m",
			topic:     "m.workspace.c.channel.sub.topic",
			shouldErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workspace, channel, subtopic, err := parseMQTTTopic(tc.prefix, tc.topic)
			if tc.shouldErr {
				if err == nil {
					t.Fatal("expected parse error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if workspace != tc.workspace || channel != tc.channel || subtopic != tc.subtopic {
				t.Fatalf("parsed topic mismatch: got workspace=%q channel=%q subtopic=%q", workspace, channel, subtopic)
			}
		})
	}
}

func TestParseMQTTTopicFromStreamRoutingKey(t *testing.T) {
	// Stream queue routing keys have the format "$queue/<prefix>/<workspace>/c/<channel>[/<subtopic>]".
	// After stripping "$queue/", the remainder is a valid MQTT-style topic for parseMQTTTopic.
	cases := []struct {
		name       string
		routingKey string
		prefix     string
		workspace  string
		channel    string
		subtopic   string
	}{
		{
			name:       "writers queue with subtopic",
			routingKey: "$queue/writers/workspace/c/channel/temp",
			prefix:     "writers",
			workspace:  "workspace",
			channel:    "channel",
			subtopic:   "temp",
		},
		{
			name:       "main queue without subtopic",
			routingKey: "$queue/m/workspace/c/channel",
			prefix:     "m",
			workspace:  "workspace",
			channel:    "channel",
			subtopic:   "",
		},
		{
			name:       "alarms queue with nested subtopic",
			routingKey: "$queue/alarms/dom/c/ch/critical/high",
			prefix:     "alarms",
			workspace:  "dom",
			channel:    "ch",
			subtopic:   "critical/high",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mqttTopic := strings.TrimPrefix(tc.routingKey, "$queue/")
			workspace, channel, subtopic, err := parseMQTTTopic(tc.prefix, mqttTopic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if workspace != tc.workspace || channel != tc.channel || subtopic != tc.subtopic {
				t.Fatalf("got workspace=%q channel=%q subtopic=%q", workspace, channel, subtopic)
			}
		})
	}
}

func TestStringHeader(t *testing.T) {
	headers := map[string]any{
		"external_id": "pub-1",
		"number":      42,
		"bytes":       []byte("bin"),
	}
	if got := stringHeader(headers, "external_id"); got != "pub-1" {
		t.Fatalf("expected pub-1, got %q", got)
	}
	if got := stringHeader(headers, "bytes"); got != "bin" {
		t.Fatalf("expected bin, got %q", got)
	}
	if got := stringHeader(headers, "number"); got != "" {
		t.Fatalf("expected empty for non-string, got %q", got)
	}
	if got := stringHeader(headers, "missing"); got != "" {
		t.Fatalf("expected empty for missing key, got %q", got)
	}
	if got := stringHeader(nil, "any"); got != "" {
		t.Fatalf("expected empty for nil headers, got %q", got)
	}
}

func TestFormatConsumerName(t *testing.T) {
	got := formatConsumerName("m/workspace/c/channel/#", "re/service 1")
	want := "m_workspace_c_channel__-re_service_1"
	if got != want {
		t.Fatalf("consumer name mismatch: got %q, want %q", got, want)
	}
}
