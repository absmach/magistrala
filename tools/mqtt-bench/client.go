// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/cisco/senml"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	mat "gonum.org/v1/gonum/mat"
	stat "gonum.org/v1/gonum/stat"
)

// Client - represents mqtt client
type Client struct {
	ID         string
	BrokerURL  string
	BrokerUser string
	BrokerPass string
	MsgTopic   string
	Message    func(cid string, time float64, f func() *senml.SenML) ([]byte, error)
	GetSenML   func() *senml.SenML
	MsgSize    int
	MsgCount   int
	MsgQoS     byte
	Quiet      bool
	mqttClient *mqtt.Client
	MTLS       bool
	SkipTLSVer bool
	Retain     bool
	CA         []byte
	ClientCert tls.Certificate
	ClientKey  *rsa.PrivateKey
}

type messagePayload struct {
	ID      string
	Sent    time.Time
	Payload interface{}
}

type message struct {
	ID             string
	Topic          string
	QoS            byte
	Payload        messagePayload
	Sent           time.Time
	Delivered      time.Time
	DeliveredToSub time.Time
	Error          bool
}

// Publisher
func (c *Client) runPublisher(r chan *runResults) {
	newMsgs := make(chan *message)
	pubMsgs := make(chan *message)
	doneGen := make(chan bool)
	donePub := make(chan bool)
	runResults := new(runResults)
	Inf := float64(math.Inf(+1))
	var diff float64

	// Start generator
	go c.generate(newMsgs, doneGen)

	started := time.Now()
	// Start publisher
	go c.publish(newMsgs, pubMsgs, doneGen, donePub)
	times := []float64{}
	for {
		select {
		case m := <-pubMsgs:
			cid := m.ID
			if m.Error {
				runResults.Failures++
				diff = Inf
			} else {
				runResults.Successes++
				diff = float64(m.Delivered.Sub(m.Sent).Nanoseconds() / 1000) // in microseconds
			}
			runResults.ID = cid
			times = append(times, diff)
		case <-donePub:
			// Calculate results
			duration := time.Now().Sub(started)
			timeMatrix := mat.NewDense(1, len(times), times)
			runResults.MsgTimeMin = mat.Min(timeMatrix)
			runResults.MsgTimeMax = mat.Max(timeMatrix)
			runResults.MsgTimeMean = stat.Mean(times, nil)
			runResults.MsgTimeStd = stat.StdDev(times, nil)
			runResults.RunTime = duration.Seconds()
			runResults.MsgsPerSec = float64(runResults.Successes) / duration.Seconds()

			// Report results and exit
			r <- runResults
			return
		}
	}
}

// Subscriber
func (c *Client) runSubscriber(wg *sync.WaitGroup, tot int, donePub *chan bool, res *chan *map[string](*[]float64)) {
	c.subscribe(wg, tot, donePub, res)
}

func (c *Client) generate(ch chan *message, done chan bool) {
	for i := 0; i < c.MsgCount; i++ {
		msgPayload := messagePayload{Payload: c.Message}
		ch <- &message{
			Topic:   c.MsgTopic,
			QoS:     c.MsgQoS,
			Payload: msgPayload,
		}
	}

	done <- true
	return
}

func (c *Client) subscribe(wg *sync.WaitGroup, tot int, donePub *chan bool, res *chan *map[string](*[]float64)) {
	clientID := fmt.Sprintf("sub-%v-%v", time.Now().Format(time.RFC3339Nano), c.ID)
	c.ID = clientID
	subsResults := make(map[string](*[]float64), 1)
	i := 1
	a := []float64{}

	go func() {
		for {
			select {
			case <-*donePub:
				time.Sleep(2 * time.Second)
				subsResults[c.MsgTopic] = &a
				*res <- &subsResults
				return
			}
		}
	}()

	onConnected := func(client mqtt.Client) {
		wg.Done()
		if !c.Quiet {
			log.Printf("Client %v is connected to the broker %v\n", clientID, c.BrokerURL)
		}
	}
	connLost := func(client mqtt.Client, reason error) {
		log.Printf("Client %v had lost connection to the broker: %s\n", c.ID, reason.Error())
	}
	if c.connect(onConnected, connLost) != nil {
		wg.Done()
		log.Printf("Client %v failed connecting to the broker\n", c.ID)
	}

	token := (*c.mqttClient).Subscribe(c.MsgTopic, c.MsgQoS, func(cl mqtt.Client, msg mqtt.Message) {

		arrival := float64(time.Now().UnixNano())
		var timeSent float64

		if c.GetSenML() != nil {
			mp, err := senml.Decode(msg.Payload(), senml.JSON)
			if err != nil && !c.Quiet {
				log.Printf("Failed to decode message %s\n", err.Error())
			}
			timeSent = *mp.Records[0].Value
		} else {
			tst := testMsg{}
			json.Unmarshal(msg.Payload(), &tst)
			timeSent = tst.Sent
		}

		a = append(a, (arrival - timeSent))
		i++
		if i == tot {
			subsResults[c.MsgTopic] = &a
			*res <- &subsResults
		}

	})
	token.Wait()
}

func (c *Client) publish(in, out chan *message, doneGen chan bool, donePub chan bool) {
	clientID := fmt.Sprintf("pub-%v-%v", time.Now().Format(time.RFC3339Nano), c.ID)
	c.ID = clientID
	ctr := 1
	onConnected := func(client mqtt.Client) {
		if !c.Quiet {
			log.Printf("Client %v is connected to the broker %v\n", clientID, c.BrokerURL)
		}
		for {
			select {
			case m := <-in:
				m.Sent = time.Now()
				m.ID = clientID
				pload, err := c.Message(m.ID, float64(m.Sent.UnixNano()), c.GetSenML)
				if err != nil {
					log.Printf("Failed to marshal payload - %s", err.Error())
				}
				token := client.Publish(m.Topic, m.QoS, c.Retain, pload)
				token.Wait()
				if token.Error() != nil {
					m.Error = true
				} else {
					m.Delivered = time.Now()
					m.Error = false
				}
				out <- m

				if ctr > 0 && ctr%100 == 0 {
					if !c.Quiet {
						log.Printf("Client %v published %v messages and keeps publishing...\n", clientID, ctr)
					}
				}
				ctr++
			case <-doneGen:
				donePub <- true
				if !c.Quiet {
					log.Printf("Client %v is done publishing\n", clientID)
				}
				return
			}
		}
	}
	connLost := func(client mqtt.Client, reason error) {
		log.Printf("Client %v had lost connection to the broker: %s\n", c.ID, reason.Error())
		if ctr < c.MsgCount {
			flushMessages := make([]message, c.MsgCount-ctr)
			for _, m := range flushMessages {
				m.Error = true
				out <- &m
			}
		}
		donePub <- true
	}

	if c.connect(onConnected, connLost) != nil {
		log.Printf("Failed to connect %s\n", c.ID)
		flushMessages := make([]message, c.MsgCount-ctr)
		for _, m := range flushMessages {
			m.Error = true
			out <- &m
		}
		donePub <- true
	}

}

func (c *Client) connect(onConnected func(client mqtt.Client), connLost func(client mqtt.Client, reason error)) error {
	opts := mqtt.NewClientOptions().
		AddBroker(c.BrokerURL).
		SetClientID(c.ID).
		SetCleanSession(true).
		SetAutoReconnect(false).
		SetOnConnectHandler(onConnected).
		SetConnectionLostHandler(connLost)

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
		return
	}

	if err != nil {
		log.Fatalf("Error: %s\n", err.Error())
		return
	}

	log.Printf("Connection to %s://%s:%s looks OK\n", network, host, port)
}
