// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux/pkg/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// Publisher wraps messaging Publisher exposing
// Close() method for RabbitMQ connection.
type Publisher interface {
	messaging.Publisher
	Close()
}

// NewPublisher returns RabbitMQ message Publisher.
func NewPublisher(url string) (Publisher, error) {
	endpoint := fmt.Sprintf("amqp://%s", url)
	conn, err := amqp.Dial(endpoint)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchangeName, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return nil, err
	}
	ret := &publisher{
		conn: conn,
		ch:   ch,
	}
	return ret, nil
}

func (pub *publisher) Publish(topic string, msg messaging.Message) error {
	if topic == "" {
		return ErrEmptyTopic
	}
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, topic)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
	}
	err = pub.ch.Publish(
		exchangeName,
		subject,
		false,
		false,
		amqp.Publishing{
			Headers:     amqp.Table{},
			ContentType: "application/octet-stream",
			AppId:       "mainflux-publisher",
			Body:        data,
		})

	if err != nil {
		return err
	}

	return nil
}

func (pub *publisher) Close() {
	pub.conn.Close()
}
