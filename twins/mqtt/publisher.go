// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/logger"
)

// Mqtt stores mqtt client and topic
type Mqtt struct {
	client    mqtt.Client
	channelID string
}

// New instantiates the mqtt service.
func New(mc mqtt.Client, channelID string) Mqtt {
	return Mqtt{
		client:    mc,
		channelID: channelID,
	}
}

// Connect to MQTT broker
func Connect(mqttURL, id, key string, logger logger.Logger) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttURL)
	opts.SetClientID("twins")
	opts.SetUsername(id)
	opts.SetPassword(key)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		logger.Info("Connected to MQTT broker")
	})
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		logger.Error(fmt.Sprintf("MQTT connection lost: %s", err.Error()))
		os.Exit(1)
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logger.Error(fmt.Sprintf("Failed to connect to MQTT broker: %s", token.Error()))
		os.Exit(1)
	}

	return client
}

func (m *Mqtt) Channel() string {
	return m.channelID
}

func (m *Mqtt) publish(twinID, crudOp string, payload *[]byte) error {
	topic := fmt.Sprintf("channels/%s/messages/%s/%s", m.channelID, twinID, crudOp)
	if len(twinID) < 1 {
		topic = fmt.Sprintf("channels/%s/messages/%s", m.channelID, crudOp)
	}

	token := m.client.Publish(topic, 0, false, *payload)
	token.Wait()

	return token.Error()
}

// Publish sends mqtt message to a predefined topic
func (m *Mqtt) Publish(twinID *string, err *error, succOp, failOp string, payload *[]byte) error {
	op := succOp
	if *err != nil {
		op = failOp
		esb := []byte((*err).Error())
		payload = &esb
	}

	if err := m.publish(*twinID, op, payload); err != nil {
		return err
	}

	return nil
}
