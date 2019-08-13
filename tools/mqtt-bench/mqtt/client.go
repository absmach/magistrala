package mqtt

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/tools/mqtt-bench/res"
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
	MsgSize    int
	MsgCount   int
	MsgQoS     byte
	Quiet      bool
	mqttClient *mqtt.Client
	Mtls       bool
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

// RunPublisher - runs publisher
func (c *Client) RunPublisher(r chan *res.RunResults, mtls bool) {
	newMsgs := make(chan *message)
	pubMsgs := make(chan *message)
	doneGen := make(chan bool)
	donePub := make(chan bool)
	runResults := new(res.RunResults)

	started := time.Now()
	// Start generator
	go c.genMessages(newMsgs, doneGen)
	// Start publisher
	go c.pubMessages(newMsgs, pubMsgs, doneGen, donePub, mtls)

	times := []float64{}

	for {
		select {
		case m := <-pubMsgs:
			cid := m.ID
			if m.Error {
				runResults.Failures++
			} else {
				runResults.Successes++
				runResults.ID = cid
				times = append(times, float64(m.Delivered.Sub(m.Sent).Nanoseconds()/1000)) // in microseconds
			}
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

// RunSubscriber - runs a subscriber
func (c *Client) RunSubscriber(wg *sync.WaitGroup, subTimes *res.SubTimes, done *chan bool, mtls bool) {
	defer wg.Done()
	// Start subscriber
	c.subscribe(wg, subTimes, done, mtls)

}

func (c *Client) genMessages(ch chan *message, done chan bool) {
	for i := 0; i < c.MsgCount; i++ {
		msgPayload := messagePayload{Payload: make([]byte, c.MsgSize)}
		ch <- &message{
			Topic:   c.MsgTopic,
			QoS:     c.MsgQoS,
			Payload: msgPayload,
		}
	}
	done <- true
	return
}

func (c *Client) subscribe(wg *sync.WaitGroup, subTimes *res.SubTimes, done *chan bool, mtls bool) {
	clientID := fmt.Sprintf("sub-%v-%v", time.Now().Format(time.RFC3339Nano), c.ID)
	c.ID = clientID

	onConnected := func(client mqtt.Client) {
		if !c.Quiet {
			log.Printf("CLIENT %v is connected to the broker %v\n", clientID, c.BrokerURL)
		}
	}

	c.connect(onConnected, mtls)

	token := (*c.mqttClient).Subscribe(c.MsgTopic, 0, func(cl mqtt.Client, msg mqtt.Message) {

		mp := messagePayload{}
		err := json.Unmarshal(msg.Payload(), &mp)
		if err != nil {
			log.Printf("CLIENT %s failed to decode message", clientID)
		}
	})

	token.Wait()

}

func (c *Client) pubMessages(in, out chan *message, doneGen chan bool, donePub chan bool, mtls bool) {
	clientID := fmt.Sprintf("pub-%v-%v", time.Now().Format(time.RFC3339Nano), c.ID)
	c.ID = clientID
	onConnected := func(client mqtt.Client) {
		if !c.Quiet {
			log.Printf("CLIENT %v is connected to the broker %v\n", clientID, c.BrokerURL)
		}
		ctr := 0
		for {
			select {
			case m := <-in:
				m.Sent = time.Now()
				m.ID = clientID
				m.Payload.ID = clientID
				m.Payload.Sent = m.Sent

				pload, err := json.Marshal(m.Payload)
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
						log.Printf("CLIENT %v published %v messages and keeps publishing...\n", clientID, ctr)
					}
				}
				ctr++
			case <-doneGen:
				donePub <- true
				if !c.Quiet {
					log.Printf("CLIENT %v is done publishing\n", clientID)
				}
				return
			}
		}
	}

	c.connect(onConnected, mtls)

}

func (c *Client) connect(onConnected func(client mqtt.Client), mtls bool) error {
	opts := mqtt.NewClientOptions().
		AddBroker(c.BrokerURL).
		SetClientID(c.ID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(onConnected).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			log.Printf("CLIENT %s lost connection to the broker: %v. Will reconnect...\n", c.ID, reason.Error())
		})
	if c.BrokerUser != "" && c.BrokerPass != "" {
		opts.SetUsername(c.BrokerUser)
		opts.SetPassword(c.BrokerPass)
	}
	if mtls {

		cfg := &tls.Config{
			InsecureSkipVerify: c.SkipTLSVer,
		}

		if c.CA != nil {
			cfg.RootCAs = x509.NewCertPool()
			if cfg.RootCAs.AppendCertsFromPEM(c.CA) {
				log.Printf("Successfully added certificate\n")
			}
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
		log.Printf("CLIENT %v had error connecting to the broker: %s\n", c.ID, token.Error())
		return token.Error()
	}
	return nil
}
