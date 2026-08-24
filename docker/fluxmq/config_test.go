// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq_test

import (
	"os"
	"slices"
	"testing"

	"github.com/absmach/fluxmq/topics"
	"gopkg.in/yaml.v3"
)

type brokerConfig struct {
	Queues []struct {
		Name   string   `yaml:"name"`
		Topics []string `yaml:"topics"`
	} `yaml:"queues"`
}

func TestQueueBindingsDoNotOverlap(t *testing.T) {
	testCases := []struct {
		topic string
		want  []string
	}{
		{topic: "$queue/mqtt/client", want: []string{"mqtt"}},
		{topic: "$queue/events/workspace/client", want: []string{"events"}},
		{topic: "$queue/writers/workspace/channel", want: []string{"writers"}},
		{topic: "$queue/alarms/workspace/channel", want: []string{"alarms"}},
		// A queue addressed with no trailing path still has to land in its own
		// stream: pkg/events/fluxmq addresses exactly "$queue/events" when the
		// stream name resolves to an empty path, and a binding that stopped
		// matching its own parent level would drop those publications with no
		// error, since an unmatched topic capture is not a failure.
		{topic: "$queue/events", want: []string{"events"}},
		// Channel messages reach stream "m" through its own "m/#" binding, so
		// nothing addresses the queue directly. Were that to change, the
		// publication would match no queue at all rather than fall through to
		// the reserved one.
		{topic: "$queue/m/workspace/channel", want: nil},
	}

	for _, configFile := range []string{"node1.yaml", "node2.yaml", "node3.yaml"} {
		data, err := os.ReadFile(configFile)
		if err != nil {
			t.Fatalf("read %s: %v", configFile, err)
		}
		var cfg brokerConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("parse %s: %v", configFile, err)
		}

		for _, tc := range testCases {
			var got []string
			for _, queue := range cfg.Queues {
				for _, pattern := range queue.Topics {
					if topics.TopicMatch(pattern, tc.topic) {
						got = append(got, queue.Name)
						break
					}
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("%s: queues matching %q = %v, want %v", configFile, tc.topic, got, tc.want)
			}
		}
	}
}
