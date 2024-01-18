// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/pelletier/go-toml"
)

// Benchmark - main benchmarking function.
func Benchmark(cfg Config) error {
	if err := checkConnection(cfg.MQTT.Broker.URL, 1); err != nil {
		return err
	}
	logger, err := mglog.New(os.Stdout, "debug")
	if err != nil {
		return err
	}

	subsResults := map[string](*[]float64){}
	var caByte []byte
	if cfg.MQTT.TLS.MTLS {
		caFile, err := os.Open(cfg.MQTT.TLS.CA)

		defer func() {
			if err = caFile.Close(); err != nil {
				logger.Warn(fmt.Sprintf("Could  not close file: %s", err))
			}
		}()
		if err != nil {
			logger.Warn(err.Error())
		}
		caByte, _ = io.ReadAll(caFile)
	}

	data, err := os.ReadFile(cfg.Mg.ConnFile)
	if err != nil {
		return fmt.Errorf("error loading connections file: %s", err)
	}

	mg := magistrala{}
	if err := toml.Unmarshal(data, &mg); err != nil {
		return fmt.Errorf("cannot load Magistrala connections config %s \nUse tools/provision to create file", cfg.Mg.ConnFile)
	}

	resCh := make(chan *runResults)
	finishedPub := make(chan bool)

	startStamp := time.Now()

	n := len(mg.Channels)
	var cert tls.Certificate

	start := time.Now()

	// Publishers
	for i := 0; i < cfg.Test.Pubs; i++ {
		mgChan := mg.Channels[i%n]
		mgThing := mg.Things[i%n]

		if cfg.MQTT.TLS.MTLS {
			cert, err = tls.X509KeyPair([]byte(mgThing.MTLSCert), []byte(mgThing.MTLSKey))
			if err != nil {
				return err
			}
		}
		c, err := makeClient(i, cfg, mgChan, mgThing, startStamp, caByte, cert)
		if err != nil {
			return fmt.Errorf("unable to create message payload %s", err.Error())
		}

		errorChan := make(chan error)
		go c.publish(resCh, errorChan)

		for {
			err := <-errorChan
			if err != nil {
				return err
			}
		}
	}

	// Collect the results
	var results []*runResults
	if cfg.Test.Pubs > 0 {
		results = make([]*runResults, cfg.Test.Pubs)
	}

	// Wait for publishers to finish
	go func() {
		for i := 0; i < cfg.Test.Pubs; i++ {
			results[i] = <-resCh
		}
		finishedPub <- true
	}()

	<-finishedPub

	totalTime := time.Since(start)
	totals := calculateTotalResults(results, totalTime, subsResults)
	if totals == nil {
		return fmt.Errorf("totals not assigned")
	}

	// Print sats
	printResults(results, totals, cfg.MQTT.Message.Format, cfg.Log.Quiet)
	return nil
}

func getBytePayload(size int, m message) (handler, error) {
	// Calculate payload size.
	var b []byte
	s, err := json.Marshal(&m)
	if err != nil {
		return nil, err
	}
	n := len(s)
	if n < size {
		sz := size - n
		for {
			b = make([]byte, sz)
			if _, err = rand.Read(b); err != nil {
				return nil, err
			}
			m.Payload = b
			content, err := json.Marshal(&m)
			if err != nil {
				return nil, err
			}
			l := len(content)
			// Use range because the size of generated JSON
			// depends on current time and random byte array.
			if l <= size+5 && l >= size-5 {
				break
			}
			if l > size {
				sz--
			}
			if l < size {
				sz++
			}
		}
	}

	ret := func(m *message) ([]byte, error) {
		m.Payload = b
		m.Sent = time.Now()
		return json.Marshal(m)
	}
	return ret, nil
}

func makeClient(i int, cfg Config, mgChan mgChannel, mgThing mgThing, start time.Time, caCert []byte, clientCert tls.Certificate) (*Client, error) {
	c := &Client{
		ID:         strconv.Itoa(i),
		BrokerURL:  cfg.MQTT.Broker.URL,
		BrokerUser: mgThing.ThingID,
		BrokerPass: mgThing.ThingKey,
		MsgTopic:   fmt.Sprintf("channels/%s/messages/%d/test", mgChan.ChannelID, start.UnixNano()),
		MsgSize:    cfg.MQTT.Message.Size,
		MsgCount:   cfg.Test.Count,
		MsgQoS:     byte(cfg.MQTT.Message.QoS),
		Quiet:      cfg.Log.Quiet,
		MTLS:       cfg.MQTT.TLS.MTLS,
		SkipTLSVer: cfg.MQTT.TLS.SkipTLSVer,
		CA:         caCert,
		timeout:    cfg.MQTT.Timeout,
		ClientCert: clientCert,
		Retain:     cfg.MQTT.Message.Retain,
	}
	msg := message{
		Topic: c.MsgTopic,
		QoS:   c.MsgQoS,
		ID:    c.ID,
		Sent:  time.Now(),
	}
	h, err := getBytePayload(cfg.MQTT.Message.Size, msg)
	if err != nil {
		return nil, err
	}

	c.SendMsg = h
	return c, nil
}
