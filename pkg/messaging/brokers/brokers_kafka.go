//go:build kafka
// +build kafka

// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package brokers

import (
	"log"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/kafka"
)

// SubjectAllChannels represents subject to subscribe for all the channels.
const SubjectAllChannels = "channels.*"

func init() {
	log.Println("The binary was build using Kafka as the message broker")
}

func NewPublisher(url string) (messaging.Publisher, error) {
	pb, err := kafka.NewPublisher(url)
	if err != nil {
		return nil, err
	}
	return pb, nil

}

func NewPubSub(url, queue string, logger logger.Logger) (messaging.PubSub, error) {
	pb, err := kafka.NewPubSub(url, queue, logger)
	if err != nil {
		return nil, err
	}
	return pb, nil
}
