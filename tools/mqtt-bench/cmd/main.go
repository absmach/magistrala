// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains the entry point of the mqtt-bench tool.
package main

import (
	"log"

	bench "github.com/absmach/magistrala/tools/mqtt-bench"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	confFile := ""
	bconf := bench.Config{}

	// Command
	rootCmd := &cobra.Command{
		Use:   "mqtt-bench",
		Short: "mqtt-bench is MQTT benchmark tool for Magistrala",
		Long: `Tool for exctensive load and benchmarking of MQTT brokers used within the Magistrala platform.
Complete documentation is available at https://docs.mainflux.io`,
		Run: func(cmd *cobra.Command, args []string) {
			if confFile != "" {
				viper.SetConfigFile(confFile)

				if err := viper.ReadInConfig(); err != nil {
					log.Printf("Failed to load config - %s", err)
				}

				if err := viper.Unmarshal(&bconf); err != nil {
					log.Printf("Unable to decode into struct, %v", err)
				}
			}

			if err := bench.Benchmark(bconf); err != nil {
				log.Fatal(err)
			}
		},
	}

	// Flags
	// MQTT Broker
	rootCmd.PersistentFlags().StringVarP(&bconf.MQTT.Broker.URL, "broker", "b", "tcp://localhost:1883",
		"address for mqtt broker, for secure use tcps and 8883")

	// MQTT Message
	rootCmd.PersistentFlags().IntVarP(&bconf.MQTT.Message.Size, "size", "z", 100, "Size of message payload bytes")
	rootCmd.PersistentFlags().StringVarP(&bconf.MQTT.Message.Payload, "payload", "l", "", "Template message")
	rootCmd.PersistentFlags().StringVarP(&bconf.MQTT.Message.Format, "format", "f", "text", "Output format: text|json")
	rootCmd.PersistentFlags().IntVarP(&bconf.MQTT.Message.QoS, "qos", "q", 0, "QoS for published messages, values 0 1 2")
	rootCmd.PersistentFlags().BoolVarP(&bconf.MQTT.Message.Retain, "retain", "r", false, "Retain mqtt messages")
	rootCmd.PersistentFlags().IntVarP(&bconf.MQTT.Timeout, "timeout", "o", 10000, "Timeout mqtt messages")

	// MQTT TLS
	rootCmd.PersistentFlags().BoolVarP(&bconf.MQTT.TLS.MTLS, "mtls", "", false, "Use mtls for connection")
	rootCmd.PersistentFlags().BoolVarP(&bconf.MQTT.TLS.SkipTLSVer, "skipTLSVer", "t", false, "Skip tls verification")
	rootCmd.PersistentFlags().StringVarP(&bconf.MQTT.TLS.CA, "ca", "", "ca.crt", "CA file")

	// Test params
	rootCmd.PersistentFlags().IntVarP(&bconf.Test.Count, "count", "n", 100, "Number of messages sent per publisher")
	rootCmd.PersistentFlags().IntVarP(&bconf.Test.Subs, "subs", "s", 10, "Number of subscribers")
	rootCmd.PersistentFlags().IntVarP(&bconf.Test.Pubs, "pubs", "p", 10, "Number of publishers")

	// Log params
	rootCmd.PersistentFlags().BoolVarP(&bconf.Log.Quiet, "quiet", "", false, "Suppress messages")

	// Config file
	rootCmd.PersistentFlags().StringVarP(&confFile, "config", "c", "config.toml", "config file for mqtt-bench")
	rootCmd.PersistentFlags().StringVarP(&bconf.Mg.ConnFile, "magistrala", "m", "connections.toml", "config file for Magistrala connections")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
