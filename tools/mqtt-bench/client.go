// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Set default ping timeout to large value, so that ping
// won't fail in the case of broker pingresp delay.
const pingTimeout = 10000

// Client - represents mqtt client
type Client struct {
	ID         string
	BrokerURL  string
	BrokerUser string
	BrokerPass string
	MsgTopic   string
	MsgSize    int
	MsgCount   int
	MsgQoS     byte
	Quiet      bool
	timeout    int
	mqttClient *mqtt.Client
	MTLS       bool
	SkipTLSVer bool
	Retain     bool
	CA         []byte
	ClientCert tls.Certificate
	ClientKey  *rsa.PrivateKey
	SendMsg    handler
}

type message struct {
	ID        string    `json:"id"`
	Topic     string    `json:"topic"`
	QoS       byte      `json:"qos"`
	Payload   []byte    `json:"payload"`
	Sent      time.Time `json:"sent"`
	Delivered time.Time `json:"delivered"`
	Error     bool      `json:"error"`
}

type handler func(*message) ([]byte, error)

func (c *Client) publish(r chan *runResults) {
	res := &runResults{}
	times := make([]*float64, c.MsgCount)

	start := time.Now()
	if c.connect() != nil {
		flushMessages := make([]message, c.MsgCount)
		for i, m := range flushMessages {
			m.Error = true
			times[i] = calcMsgRes(&m, res)
		}
		r <- calcRes(res, start, arr(times))
		return
	}
	if !c.Quiet {
		log.Printf("Client %v is connected to the broker %v\n", c.ID, c.BrokerURL)
	}
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	// Use a single message.
	m := message{
		Topic: c.MsgTopic,
		QoS:   c.MsgQoS,
		ID:    c.ID,
		Sent:  time.Now(),
	}
	payload, err := c.SendMsg(&m)
	if err != nil {
		log.Fatalf("Failed to marshal payload - %s", err.Error())
	}

	for i := 0; i < c.MsgCount; i++ {
		wg.Add(1)
		go func(mut *sync.Mutex, wg *sync.WaitGroup, i int, m message) {
			defer wg.Done()
			m.Sent = time.Now()

			token := (*c.mqttClient).Publish(m.Topic, m.QoS, c.Retain, payload)
			if !token.WaitTimeout(time.Second*time.Duration(c.timeout)) || token.Error() != nil || !(*c.mqttClient).IsConnectionOpen() {
				m.Error = true
				mu.Lock()
				times[i] = calcMsgRes(&m, res)
				mu.Unlock()
				return
			}

			m.Delivered = time.Now()
			m.Error = false
			mu.Lock()
			times[i] = calcMsgRes(&m, res)
			mu.Unlock()

			if !c.Quiet && i > 0 && i%100 == 0 {
				log.Printf("Client %v published %v messages and keeps publishing...\n", c.ID, i)
			}
		}(&mu, &wg, i, m)
	}
	wg.Wait()

	r <- calcRes(res, start, arr(times))
}

func (c *Client) connect() error {
	opts := mqtt.NewClientOptions().
		AddBroker(c.BrokerURL).
		SetClientID(c.ID).
		SetCleanSession(false).
		SetAutoReconnect(false).
		SetOnConnectHandler(c.connected).
		SetConnectionLostHandler(c.connLost).
		SetPingTimeout(time.Second * pingTimeout).
		SetAutoReconnect(true).
		SetCleanSession(false)

	if c.BrokerUser != "" && c.BrokerPass != "" {
		opts.SetUsername(c.BrokerUser)
		opts.SetPassword(c.BrokerPass)
	}

	if c.MTLS {
		cfg := &tls.Config{
			InsecureSkipVerify: c.SkipTLSVer,
		}

		if c.CA != nil {
			cfg.RootCAs = x509.NewCertPool()
			cfg.RootCAs.AppendCertsFromPEM(c.CA)
		}
		if c.ClientCert.Certificate != nil {
			cfg.Certificates = []tls.Certificate{c.ClientCert}
		}

		cfg.BuildNameToCertificate()
		opts.SetTLSConfig(cfg)
		opts.SetProtocolVersion(4)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	c.mqttClient = &client

	if token.Error() != nil {
		log.Printf("Client %v had error connecting to the broker: %s\n", c.ID, token.Error().Error())
		return token.Error()
	}

	return nil
}

func checkConnection(broker string, timeoutSecs int) {
	s := strings.Split(broker, ":")
	if len(s) != 3 {
		log.Fatalf("Wrong host address format")
	}

	network := s[0]
	host := strings.Trim(s[1], "/")
	port := s[2]

	log.Println("Testing connection...")
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), time.Duration(timeoutSecs)*time.Second)
	conClose := func() {
		if conn != nil {
			log.Println("Closing testing connection...")
			conn.Close()
		}
	}

	defer conClose()
	if err, ok := err.(*net.OpError); ok && err.Timeout() {
		log.Fatalf("Timeout error: %s\n", err.Error())
	}

	if err != nil {
		log.Fatalf("Error: %s\n", err.Error())
	}

	log.Printf("Connection to %s://%s:%s looks OK\n", network, host, port)
}

func arr(a []*float64) []float64 {
	ret := []float64{}
	for _, v := range a {
		if v != nil {
			ret = append(ret, *v)
		}
	}
	if len(ret) == 0 {
		ret = append(ret, 0)
	}
	return ret
}

func (c *Client) connected(client mqtt.Client) {
	if !c.Quiet {
		log.Printf("Client %v is connected to the broker %v\n", c.ID, c.BrokerURL)
	}
}

func (c *Client) connLost(client mqtt.Client, reason error) {
	log.Printf("Client %v had lost connection to the broker: %s\n", c.ID, reason.Error())
}
