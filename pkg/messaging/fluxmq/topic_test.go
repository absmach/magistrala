// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"strings"
	"testing"
)

func TestQueueTopic(t *testing.T) {
	got := queueTopic("m", "domain.c.channel.subtopic")
	want := "$queue/m/domain/c/channel/subtopic"
	if got != want {
		t.Fatalf("queue topic mismatch: got %q, want %q", got, want)
	}
}

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
			topic:  "m.>",
			want:   "#",
		},
		{
			name:   "all messages without explicit prefix",
			prefix: "writers",
			topic:  ">",
			want:   "#",
		},
		{
			name:   "specific topic filter",
			prefix: "writers",
			topic:  "writers.domain.c.channel.*",
			want:   "domain/c/channel/+",
		},
		{
			name:   "topic without prefix",
			prefix: "alarms",
			topic:  "domain.c.channel.>",
			want:   "domain/c/channel/#",
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
	got := queueFilter("writers", "writers.>")
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
			topic:  "m.>",
			want:   "m/#",
		},
		{
			name:   "wildcard topic",
			prefix: "writers",
			topic:  ">",
			want:   "writers/#",
		},
		{
			name:   "specific topic",
			prefix: "m",
			topic:  "m.domain.c.channel.subtopic",
			want:   "m/domain/c/channel/subtopic",
		},
		{
			name:   "single-level wildcard",
			prefix: "m",
			topic:  "m.domain.c.*.subtopic",
			want:   "m/domain/c/+/subtopic",
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
		domain    string
		channel   string
		subtopic  string
		shouldErr bool
	}{
		{
			name:     "default prefix with subtopic path",
			prefix:   "m",
			topic:    "m/domain/c/channel/sub/topic",
			domain:   "domain",
			channel:  "channel",
			subtopic: "sub.topic",
		},
		{
			name:     "alternate prefix without subtopic",
			prefix:   "writers",
			topic:    "writers/domain/c/channel",
			domain:   "domain",
			channel:  "channel",
			subtopic: "",
		},
		{
			name:     "leading slash is ignored",
			prefix:   "alarms",
			topic:    "/alarms/domain/c/channel/critical/high",
			domain:   "domain",
			channel:  "channel",
			subtopic: "critical.high",
		},
		{
			name:      "mismatched prefix",
			prefix:    "writers",
			topic:     "m/domain/c/channel",
			shouldErr: true,
		},
		{
			name:      "invalid shape",
			prefix:    "m",
			topic:     "m/domain/channel",
			shouldErr: true,
		},
		{
			name:      "empty subtopic segment",
			prefix:    "m",
			topic:     "m/domain/c/channel/sub//topic",
			shouldErr: true,
		},
		{
			name:      "dot topic is invalid",
			prefix:    "m",
			topic:     "m.domain.c.channel.sub.topic",
			shouldErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			domain, channel, subtopic, err := parseMQTTTopic(tc.prefix, tc.topic)
			if tc.shouldErr {
				if err == nil {
					t.Fatal("expected parse error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if domain != tc.domain || channel != tc.channel || subtopic != tc.subtopic {
				t.Fatalf("parsed topic mismatch: got domain=%q channel=%q subtopic=%q", domain, channel, subtopic)
			}
		})
	}
}

func TestParseMQTTTopicFromStreamRoutingKey(t *testing.T) {
	// Stream queue routing keys have the format "$queue/<prefix>/<domain>/c/<channel>[/<subtopic>]".
	// After stripping "$queue/", the remainder is a valid MQTT-style topic for parseMQTTTopic.
	cases := []struct {
		name       string
		routingKey string
		prefix     string
		domain     string
		channel    string
		subtopic   string
	}{
		{
			name:       "writers queue with subtopic",
			routingKey: "$queue/writers/domain/c/channel/temp",
			prefix:     "writers",
			domain:     "domain",
			channel:    "channel",
			subtopic:   "temp",
		},
		{
			name:       "main queue without subtopic",
			routingKey: "$queue/m/domain/c/channel",
			prefix:     "m",
			domain:     "domain",
			channel:    "channel",
			subtopic:   "",
		},
		{
			name:       "alarms queue with nested subtopic",
			routingKey: "$queue/alarms/dom/c/ch/critical/high",
			prefix:     "alarms",
			domain:     "dom",
			channel:    "ch",
			subtopic:   "critical.high",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mqttTopic := strings.TrimPrefix(tc.routingKey, "$queue/")
			domain, channel, subtopic, err := parseMQTTTopic(tc.prefix, mqttTopic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if domain != tc.domain || channel != tc.channel || subtopic != tc.subtopic {
				t.Fatalf("got domain=%q channel=%q subtopic=%q", domain, channel, subtopic)
			}
		})
	}
}

func TestStringHeader(t *testing.T) {
	headers := map[string]any{
		"publisher": "pub-1",
		"number":    42,
		"bytes":     []byte("bin"),
	}
	if got := stringHeader(headers, "publisher"); got != "pub-1" {
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
	got := formatConsumerName("m.domain.c.channel.>", "re/service 1")
	want := "m_domain_c_channel__-re_service_1"
	if got != want {
		t.Fatalf("consumer name mismatch: got %q, want %q", got, want)
	}
}
