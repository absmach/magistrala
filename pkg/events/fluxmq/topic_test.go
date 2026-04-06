// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import "testing"

func TestCanonicalStream(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   string
	}{
		{
			name:   "raw magistrala stream",
			stream: "magistrala.domain.create",
			want:   "events.magistrala.domain.create",
		},
		{
			name:   "already prefixed stream",
			stream: "events.magistrala.group.*",
			want:   "events.magistrala.group.*",
		},
		{
			name:   "all events wildcard",
			stream: ">",
			want:   "events.>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := canonicalStream(tc.stream); got != tc.want {
				t.Fatalf("canonicalStream(%q) = %q, want %q", tc.stream, got, tc.want)
			}
		})
	}
}

func TestQueueFilter(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   string
	}{
		{
			name:   "domain wildcard",
			stream: "events.magistrala.domain.*",
			want:   "$queue/events/magistrala/domain/+",
		},
		{
			name:   "all events",
			stream: ">",
			want:   "$queue/events/#",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := queueFilter(tc.stream); got != tc.want {
				t.Fatalf("queueFilter(%q) = %q, want %q", tc.stream, got, tc.want)
			}
		})
	}
}

func TestStreamFilter(t *testing.T) {
	tests := []struct {
		name   string
		stream string
		want   string
	}{
		{
			name:   "domain wildcard",
			stream: "events.magistrala.domain.*",
			want:   "magistrala/domain/+",
		},
		{
			name:   "all events",
			stream: ">",
			want:   "#",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := streamFilter(tc.stream); got != tc.want {
				t.Fatalf("streamFilter(%q) = %q, want %q", tc.stream, got, tc.want)
			}
		})
	}
}
