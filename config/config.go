/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

// Config struct
type Config struct {
	// HTTP
	HTTPHost string
	HTTPPort int

	// Mongo
	MongoHost     string
	MongoPort     int
	MongoDatabase string

	// MQTT
	MQTTHost string
	MQTTPort int

	// Influx
	InfluxHost     string
	InfluxPort     int
	InfluxDatabase string
}

// Parse TOML config
func (cfg *Config) Parse() {

	var confFile string

	testEnv := os.Getenv("TEST_ENV")
	if testEnv == "" && len(os.Args) > 1 {
		// We are not in the TEST_ENV (where different args are provided)
		// and provided config file as an argument
		confFile = os.Args[1]
	} else {
		// default cfg path to source dir, as we keep cfg.yml there
		confFile = os.Getenv("GOPATH") + "/src/github.com/mainflux/mainflux/config/config.toml"
	}

	if _, err := toml.DecodeFile(confFile, &cfg); err != nil {
		// handle error
		fmt.Println("Error parsing Toml")
	}
}
