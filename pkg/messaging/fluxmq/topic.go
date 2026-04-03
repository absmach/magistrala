// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"fmt"
	"strings"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/supermq/pkg/messaging"
)

const queuePrefix = "$queue/"

var nameReplacer = strings.NewReplacer(
	" ", "_",
	".", "_",
	"*", "_",
	">", "_",
	"/", "_",
	"\\", "_",
	"+", "_",
	"#", "_",
)

func streamFilter(prefix, topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" || topic == "#" {
		return "#"
	}

	switch {
	case topic == prefix:
		return "#"
	case strings.HasPrefix(topic, prefix+"/"):
		topic = strings.TrimPrefix(topic, prefix+"/")
	}

	topic = strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if topic == "" {
		return "#"
	}

	return topic
}

// topicFilter returns the MQTT topic filter for subscribing to regular
// (non-queued) messages. It strips the prefix and re-prepends it to
// normalize the filter.
func topicFilter(prefix, topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" || topic == "#" {
		return prefix + "/#"
	}

	switch {
	case topic == prefix:
		return prefix + "/#"
	case strings.HasPrefix(topic, prefix+"/"):
		topic = strings.TrimPrefix(topic, prefix+"/")
	}

	topic = strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if topic == "" || topic == "#" {
		return prefix + "/#"
	}

	return prefix + "/" + topic
}

func queueFilter(prefix, topic string) string {
	path := streamFilter(prefix, topic)
	if path == "#" {
		return queuePrefix + prefix + "/#"
	}

	return queuePrefix + prefix + "/" + path
}

func queueTopic(prefix, topic string) string {
	topic = strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if topic == "" {
		return queuePrefix + prefix
	}

	return queuePrefix + prefix + "/" + topic
}

func parseMQTTTopic(prefix, topic string) (domainID, channelID, subtopic string, err error) {
	topic = strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if !strings.HasPrefix(topic, prefix+"/") {
		return "", "", "", messaging.ErrMalformedTopic
	}
	// Replace the broker-specific prefix with the canonical message prefix
	// so ParseSubscribeTopic can parse it.
	normalized := msgPrefix + "/" + strings.TrimPrefix(topic, prefix+"/")

	domainID, channelID, subtopic, _, err = messaging.ParseSubscribeTopic(normalized)
	if err != nil {
		return "", "", "", err
	}

	return domainID, channelID, subtopic, nil
}

func stringHeader(headers map[string]any, key string) string {
	if headers == nil {
		return ""
	}
	v, ok := headers[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	default:
		return ""
	}
}

func declareStream(client *fluxamqp.Client, prefix string) error {
	_, err := client.DeclareStreamQueue(&fluxamqp.StreamQueueOptions{
		Name:    prefix,
		Durable: true,
	})
	return err
}

func formatConsumerName(topic, id string) string {
	// Consumer group names must avoid whitespace and wildcard/path separators.
	topic = nameReplacer.Replace(topic)
	id = nameReplacer.Replace(id)
	return fmt.Sprintf("%s-%s", topic, id)
}
