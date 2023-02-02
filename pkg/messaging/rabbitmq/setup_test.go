// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rabbitmq_test

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"

	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/rabbitmq"
	dockertest "github.com/ory/dockertest/v3"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

const (
	port          = "5672/tcp"
	brokerName    = "rabbitmq"
	brokerVersion = "3.9.20"
)

var (
	publisher messaging.Publisher
	pubsub    messaging.PubSub
	logger    mflog.Logger
	address   string
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.Run(brokerName, brokerVersion, []string{})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}
	handleInterrupt(pool, container)

	address = fmt.Sprintf("amqp://%s:%s", "localhost", container.GetPort(port))
	if err := pool.Retry(func() error {
		publisher, err = rabbitmq.NewPublisher(address)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	logger, err = mflog.New(os.Stdout, mflog.Debug.String())
	if err != nil {
		log.Fatalf(err.Error())
	}
	if err := pool.Retry(func() error {
		pubsub, err = rabbitmq.NewPubSub(address, "mainflux", logger)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()
	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}

func newConn() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(address)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}
	if err := ch.ExchangeDeclare(exchangeName, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return nil, nil, err
	}

	return conn, ch, nil
}

func rabbitHandler(deliveries <-chan amqp.Delivery, h messaging.MessageHandler) {
	for d := range deliveries {
		var msg messaging.Message
		if err := proto.Unmarshal(d.Body, &msg); err != nil {
			logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
			return
		}
		if err := h.Handle(&msg); err != nil {
			logger.Warn(fmt.Sprintf("Failed to handle Mainflux message: %s", err))
			return
		}
	}
}

func subscribe(t *testing.T, ch *amqp.Channel, topic string) <-chan amqp.Delivery {
	_, err := ch.QueueDeclare(topic, true, true, true, false, nil)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	err = ch.QueueBind(topic, topic, exchangeName, false, nil)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	clientID := fmt.Sprintf("%s-%s", topic, clientID)
	msgs, err := ch.Consume(topic, clientID, true, false, false, false, nil)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	return msgs
}

func handleInterrupt(pool *dockertest.Pool, container *dockertest.Resource) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if err := pool.Purge(container); err != nil {
			log.Fatalf("Could not purge container: %s", err)
		}
		os.Exit(0)
	}()
}
