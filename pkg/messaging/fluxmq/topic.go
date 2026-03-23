// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"fmt"
	"strings"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
)

const queuePrefix = "$queue/"

var (
	topicReplacer = strings.NewReplacer(".", "/", "*", "+", ">", "#")
	nameReplacer  = strings.NewReplacer(
		" ", "_",
		".", "_",
		"*", "_",
		">", "_",
		"/", "_",
		"\\", "_",
	)
)

func canonicalPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return msgPrefix
	}
	return prefix
}

func streamQueue(prefix string) string {
	return canonicalPrefix(prefix)
}

func brokerPath(topic string) string {
	topic = strings.TrimSpace(topic)
	topic = strings.TrimPrefix(topic, ".")
	if topic == "" {
		return ""
	}

	return topicReplacer.Replace(topic)
}

func streamFilter(prefix, topic string) string {
	path := filterPath(prefix, topic)
	if path == "" {
		return "#"
	}
	return path
}

func queueFilter(prefix, topic string) string {
	queue := streamQueue(prefix)
	path := streamFilter(prefix, topic)
	if path == "#" {
		return queuePrefix + queue + "/#"
	}

	return queuePrefix + queue + "/" + path
}

func queueTopic(prefix, topic string) string {
	queue := streamQueue(prefix)
	path := brokerPath(topic)
	if path == "" {
		return queuePrefix + queue
	}

	return queuePrefix + queue + "/" + path
}

func filterPath(prefix, topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" || topic == ">" {
		return "#"
	}

	prefix = canonicalPrefix(prefix)
	switch {
	case topic == prefix:
		topic = ">"
	case strings.HasPrefix(topic, prefix+"."):
		topic = strings.TrimPrefix(topic, prefix+".")
	}

	return brokerPath(topic)
}

func formatConsumerName(topic, id string) string {
	// Consumer group names must avoid whitespace and wildcard/path separators.
	topic = nameReplacer.Replace(topic)
	id = nameReplacer.Replace(id)
	return fmt.Sprintf("%s-%s", topic, id)
}

func declareStream(client *fluxamqp.Client, prefix string) error {
	_, err := client.DeclareStreamQueue(&fluxamqp.StreamQueueOptions{
		Name:    streamQueue(prefix),
		Durable: true,
	})
	return err
}
