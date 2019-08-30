// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package bench

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// Keep struct names exported, otherwise Viper unmarshaling won't work
type mqttBrokerConfig struct {
	URL string `toml:"url" mapstructure:"url"`
}

type mqttMessageConfig struct {
	Size   int    `toml:"size" mapstructure:"size"`
	Format string `toml:"format" mapstructure:"format"`
	QoS    int    `toml:"qos" mapstructure:"qos"`
	Retain bool   `toml:"retain" mapstructure:"retain"`
}

type mqttTLSConfig struct {
	MTLS       bool   `toml:"mtls" mapstructure:"mtls"`
	SkipTLSVer bool   `toml:"skiptlsver" mapstructure:"skiptlsver"`
	CA         string `toml:"ca" mapstructure:"ca"`
}

type mqttConfig struct {
	Broker  mqttBrokerConfig  `toml:"broker" mapstructure:"broker"`
	Message mqttMessageConfig `toml:"message" mapstructure:"message"`
	TLS     mqttTLSConfig     `toml:"tls" mapstructure:"tls"`
}

type testConfig struct {
	Count int `toml:"count" mapstructure:"count"`
	Pubs  int `toml:"pubs" mapstructure:"pubs"`
	Subs  int `toml:"subs" mapstructure:"subs"`
}

type logConfig struct {
	Quiet bool `toml:"quiet" mapstructure:"quiet"`
}

type mainfluxFile struct {
	ConnFile string `toml:"connections_file" mapstructure:"connections_file"`
}

type mfConn struct {
	ChannelID string `toml:"channel_id" mapstructure:"channel_id"`
	ThingID   string `toml:"thing_id" mapstructure:"thing_id"`
	ThingKey  string `toml:"thing_key" mapstructure:"thing_key"`
	MTLSCert  string `toml:"mtls_cert" mapstructure:"mtls_cert"`
	MTLSKey   string `toml:"mtls_key" mapstructure:"mtls_key"`
}

type mainflux struct {
	Conns []mfConn `toml:"mainflux" mapstructure:"mainflux"`
}

// Config struct holds benchmark configuration
type Config struct {
	MQTT mqttConfig   `toml:"mqtt" mapstructure:"mqtt"`
	Test testConfig   `toml:"test" mapstructure:"test"`
	Log  logConfig    `toml:"log" mapstructure:"log"`
	Mf   mainfluxFile `toml:"mainflux" mapstructure:"mainflux"`
}

// JSONResults are used to export results as a JSON document
type JSONResults struct {
	Runs   []*RunResults `json:"runs"`
	Totals *TotalResults `json:"totals"`
}

// Benchmark - main benckhmarking function
func Benchmark(cfg Config) {
	var wg sync.WaitGroup
	var err error

	checkConnection(cfg.MQTT.Broker.URL, 1)
	subTimes := make(SubTimes)
	var caByte []byte
	if cfg.MQTT.TLS.MTLS {
		caFile, err := os.Open(cfg.MQTT.TLS.CA)
		defer caFile.Close()
		if err != nil {
			fmt.Println(err)
		}

		caByte, _ = ioutil.ReadAll(caFile)
	}

	payload := string(make([]byte, cfg.MQTT.Message.Size))

	mf := mainflux{}
	if _, err := toml.DecodeFile(cfg.Mf.ConnFile, &mf); err != nil {
		log.Fatalf("Cannot load Mainflux connections config %s \nuse tools/provision to create file", cfg.Mf.ConnFile)
	}

	resCh := make(chan *RunResults)
	done := make(chan bool)

	start := time.Now()
	n := len(mf.Conns)
	var cert tls.Certificate

	// Subscribers
	for i := 0; i < cfg.Test.Subs; i++ {
		mfConn := mf.Conns[i%n]

		if cfg.MQTT.TLS.MTLS {
			cert, err = tls.X509KeyPair([]byte(mfConn.MTLSCert), []byte(mfConn.MTLSKey))
			if err != nil {
				log.Fatal(err)
			}
		}

		c := &Client{
			ID:         strconv.Itoa(i),
			BrokerURL:  cfg.MQTT.Broker.URL,
			BrokerUser: mfConn.ThingID,
			BrokerPass: mfConn.ThingKey,
			MsgTopic:   fmt.Sprintf("channels/%s/messages/test", mfConn.ChannelID),
			MsgSize:    cfg.MQTT.Message.Size,
			MsgCount:   cfg.Test.Count,
			MsgQoS:     byte(cfg.MQTT.Message.QoS),
			Quiet:      cfg.Log.Quiet,
			MTLS:       cfg.MQTT.TLS.MTLS,
			SkipTLSVer: cfg.MQTT.TLS.SkipTLSVer,
			CA:         caByte,
			ClientCert: cert,
			Retain:     cfg.MQTT.Message.Retain,
			Message:    payload,
		}

		wg.Add(1)

		go c.runSubscriber(&wg, &subTimes, &done)
	}

	wg.Wait()

	// Publishers
	for i := 0; i < cfg.Test.Pubs; i++ {
		mfConn := mf.Conns[i%n]

		if cfg.MQTT.TLS.MTLS {
			cert, err = tls.X509KeyPair([]byte(mfConn.MTLSCert), []byte(mfConn.MTLSKey))
			if err != nil {
				log.Fatal(err)
			}
		}

		c := &Client{
			ID:         strconv.Itoa(i),
			BrokerURL:  cfg.MQTT.Broker.URL,
			BrokerUser: mfConn.ThingID,
			BrokerPass: mfConn.ThingKey,
			MsgTopic:   fmt.Sprintf("channels/%s/messages/test", mfConn.ChannelID),
			MsgSize:    cfg.MQTT.Message.Size,
			MsgCount:   cfg.Test.Count,
			MsgQoS:     byte(cfg.MQTT.Message.QoS),
			Quiet:      cfg.Log.Quiet,
			MTLS:       cfg.MQTT.TLS.MTLS,
			SkipTLSVer: cfg.MQTT.TLS.SkipTLSVer,
			CA:         caByte,
			ClientCert: cert,
			Retain:     cfg.MQTT.Message.Retain,
			Message:    payload,
		}

		go c.runPublisher(resCh)
	}

	// Collect the results
	var results []*RunResults
	if cfg.Test.Pubs > 0 {
		results = make([]*RunResults, cfg.Test.Pubs)
	}

	for i := 0; i < cfg.Test.Pubs; i++ {
		results[i] = <-resCh
	}

	totalTime := time.Now().Sub(start)
	totals := calculateTotalResults(results, totalTime, &subTimes)
	if totals == nil {
		return
	}

	// Print sats
	printResults(results, totals, cfg.MQTT.Message.Format, cfg.Log.Quiet)
}
