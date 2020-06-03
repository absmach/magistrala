// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"errors"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	protocol = "mqtt"
	id       = "mqtt-publisher"
	qos      = 1
)

var errConnect = errors.New("failed to connect to MQTT broker")

func newClient(address string, timeout time.Duration) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(address).
		SetUsername(id).
		SetPassword(id).
		SetClientID(id).
		SetCleanSession(false)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Error() != nil {
		return nil, token.Error()
	}

	ok := token.WaitTimeout(timeout)
	if ok && token.Error() != nil {
		return nil, token.Error()
	}
	if !ok {
		return nil, errConnect
	}

	return client, nil
}
