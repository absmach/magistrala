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
	"github.com/spf13/viper"
	"os"
)

type Config struct {
	// HTTP
	HttpHost string
	HttpPort int

	// Mongo
	MongoHost     string
	MongoPort     int
	MongoDatabase string

	// Influx
	InfluxHost     string
	InfluxPort     int
	InfluxDatabase string
}


func (this *Config) Parse() {
	/**
	 * Config
	 */
	/** Viper setup */
	viper.SetConfigType("yaml")   // or viper.SetConfigType("YAML")

	testEnv := os.Getenv("TEST_ENV")
	if testEnv == "" && len(os.Args) > 1 {
		// We are not in the TEST_ENV (where different args are provided)
		// and provided config file as an argument
		viper.SetConfigFile(os.Args[1])
	} else {
		// default cfg path to source dir, as we keep cfg.yml there
		cfgDir := os.Getenv("GOPATH") + "/src/github.com/mainflux/mainflux-lite/config"
		viper.SetConfigName("config") // name of config file (without extension)
		viper.AddConfigPath(cfgDir)   // path to look for the config file in
	}

	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	this.MongoHost = viper.GetString("mongo.host")
	this.MongoPort = viper.GetInt("mongo.port")
	this.MongoDatabase = viper.GetString("mongo.db")

	this.InfluxHost = viper.GetString("influx.host")
	this.InfluxPort = viper.GetInt("influx.port")
	this.InfluxDatabase = viper.GetString("influx.db")

	this.HttpHost = viper.GetString("http.host")
	this.HttpPort = viper.GetInt("http.port")
}
