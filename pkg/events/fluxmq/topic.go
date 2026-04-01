// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"strings"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
)

func canonicalStream(stream string) string {
	stream = strings.TrimSpace(stream)
	if stream == "" {
		return eventsPrefix
	}
	if strings.HasPrefix(stream, eventsPrefix) {
		return stream
	}
	return eventsPrefix + stream
}

func queueTopic(stream string) string {
	path := brokerPath(stream)
	if path == "" {
		return queuePrefix + eventsQueue
	}
	return queuePrefix + eventsQueue + "/" + path
}

func queueFilter(stream string) string {
	path := brokerPath(stream)
	if path == "" || path == "#" {
		return queuePrefix + eventsQueue + "/#"
	}
	return queuePrefix + eventsQueue + "/" + path
}

func streamFilter(stream string) string {
	return brokerPath(stream)
}

func brokerPath(stream string) string {
	stream = strings.TrimSpace(stream)
	stream = strings.TrimPrefix(stream, eventsPrefix)
	if stream == "" {
		return ""
	}

	replacer := strings.NewReplacer(".", "/", "*", "+", ">", "#")
	return replacer.Replace(stream)
}

func declareEventsStream(client *fluxamqp.Client) error {
	_, err := client.DeclareStreamQueue(&fluxamqp.StreamQueueOptions{
		Name:    eventsQueue,
		Durable: true,
		MaxAge:  "30D",
	})
	return err
}
