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
	"sync"
	"time"

	smqlog "github.com/absmach/supermq/logger"
	"github.com/pelletier/go-toml"
)

// Benchmark - main benchmarking function.
func Benchmark(cfg Config) error {
	if err := checkConnection(cfg.MQTT.Broker.URL, 1); err != nil {
		return err
	}
	logger, err := smqlog.New(os.Stdout, "debug")
	if err != nil {
		return err
	}

	subsResults := map[string](*[]float64){}
	var caByte []byte
	if cfg.MQTT.TLS.MTLS {
		caFile, err := os.Open(cfg.MQTT.TLS.CA)

		defer func() {
			if err = caFile.Close(); err != nil {
				logger.Warn(fmt.Sprintf("Could not close file: %s", err))
			}
		}()
		if err != nil {
			logger.Warn(err.Error())
		}
		caByte, _ = io.ReadAll(caFile)
	}

	data, err := os.ReadFile(cfg.Smq.ConnFile)
	if err != nil {
		return fmt.Errorf("error loading connections file: %s", err)
	}

	mg := superMQ{}
	if err := toml.Unmarshal(data, &mg); err != nil {
		return fmt.Errorf("cannot load SuperMQ connections config %s \nUse tools/provision to create file", cfg.Smq.ConnFile)
	}

	resCh := make(chan *runResults)
	finishedPub := make(chan bool)

	startStamp := time.Now()

	n := len(mg.Channels)
	var cert tls.Certificate

	start := time.Now()

	var wg sync.WaitGroup
	errorChan := make(chan error, cfg.Test.Pubs)

	for i := 0; i < cfg.Test.Pubs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mgChan := mg.Channels[i%n]
			mgCli := mg.Clients[i%n]

			if cfg.MQTT.TLS.MTLS {
				cert, err = tls.X509KeyPair([]byte(mgCli.MTLSCert), []byte(mgCli.MTLSKey))
				if err != nil {
					errorChan <- err
					return
				}
			}
			c, err := makeClient(i, cfg, mgChan, mgCli, startStamp, caByte, cert)
			if err != nil {
				errorChan <- fmt.Errorf("unable to create message payload %s", err.Error())
				return
			}

			c.publish(resCh, errorChan)
		}(i)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	for err := range errorChan {
		if err != nil {
			return err
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

func makeClient(i int, cfg Config, mgChan channel, cli client, start time.Time, caCert []byte, clientCert tls.Certificate) (*Client, error) {
	c := &Client{
		ID:         strconv.Itoa(i),
		BrokerURL:  cfg.MQTT.Broker.URL,
		BrokerUser: cli.ClientID,
		BrokerPass: cli.ClientSecret,
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
