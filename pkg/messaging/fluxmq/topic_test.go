// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import "testing"

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

func TestFormatConsumerName(t *testing.T) {
	got := formatConsumerName("m.domain.c.channel.>", "re/service 1")
	want := "m_domain_c_channel__-re_service_1"
	if got != want {
		t.Fatalf("consumer name mismatch: got %q, want %q", got, want)
	}
}
