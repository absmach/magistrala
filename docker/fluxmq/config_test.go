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
		{topic: "$queue/events/domain/client", want: []string{"events"}},
		{topic: "$queue/writers/domain/channel", want: []string{"writers"}},
		{topic: "$queue/alarms/domain/channel", want: []string{"alarms"}},
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
