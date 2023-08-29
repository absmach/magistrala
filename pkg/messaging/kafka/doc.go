// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package kafka holds the implementation of the Publisher and PubSub
// interfaces for the Kafka messaging system, the internal messaging
// broker of the Mainflux IoT platform. Due to the practical requirements
// implementation Publisher is created alongside PubSub. The reason for
// this is that Subscriber implementation of Kafka brings the burden of
// additional struct fields which are not used by Publisher. Subscriber
// is not implemented separately because PubSub can be used where Subscriber is needed.
//
// The publisher implementation is based on the segmentio/kafka-go library.
// Publishing messages is well supported by the library, but subscribing
// to topics is not. The library does not provide a way to subscribe to
// all topics, but only to a specific topic. This is a problem because
// the Mainflux platform uses a topic per channel, and the number of
// channels is not known in advance. The solution is to use the Zookeeper
// library to get a list of all topics and then subscribe to each of them.
// The list of topics is obtained by connecting to the Zookeeper server
// and reading the list of topics from the /brokers/topics node. The
// first message published from the topic can be lost if subscription
// happens closely followed by publishing. After the subscription, we
// guarantee that all messages will be received.
package kafka
