// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"
	"io/ioutil"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
	"github.com/pelletier/go-toml"
)

// ServiceConf represents service config.
type ServiceConf struct {
	Port           string `toml:"port"`
	LogLevel       string `toml:"log_level"`
	TLS            bool   `toml:"tls"`
	ServerCert     string `toml:"server_cert"`
	ServerKey      string `toml:"server_key"`
	ThingsURL      string `toml:"things_url"`
	UsersURL       string `toml:"users_url"`
	HTTPPort       string `toml:"http_port"`
	MfUser         string `toml:"mf_user"`
	MfPass         string `toml:"mf_pass"`
	MfAPIKey       string `toml:"mf_api_key"`
	MfBSURL        string `toml:"mf_bs_url"`
	MfWhiteListURL string `toml:"mf_white_list"`
	MfCertsURL     string `toml:"mf_certs_url"`
}

// Bootstrap represetns the Bootstrap config
type Bootstrap struct {
	X509Provision bool                   `toml:"x509_provision"`
	Provision     bool                   `toml:"provision"`
	AutoWhiteList bool                   `toml:"autowhite_list"`
	Content       map[string]interface{} `toml:"content"`
}

// Gateway represetns the Gateway config
type Gateway struct {
	Type            string `toml:"type" json:"type"`
	ExternalID      string `toml:"external_id" json:"external_id"`
	ExternalKey     string `toml:"external_key" json:"external_key"`
	CtrlChannelID   string `toml:"ctrl_channel_id" json:"ctrl_channel_id"`
	DataChannelID   string `toml:"data_channel_id" json:"data_channel_id"`
	ExportChannelID string `toml:"export_channel_id" json:"export_channel_id"`
	CfgID           string `toml:"cfg_id" json:"cfg_id"`
}

// Cert represetns the certificate config
type Cert struct {
	TTL     string `json:"ttl" toml:"ttl"`
	KeyBits int    `json:"key_bits" toml:"key_bits"`
	KeyType string `json:"key_type"`
}

// Config struct of Provision
type Config struct {
	File      string           `toml:"file"`
	Server    ServiceConf      `toml:"server" mapstructure:"server"`
	Bootstrap Bootstrap        `toml:"bootstrap" mapstructure:"bootstrap"`
	Things    []things.Thing   `toml:"things" mapstructure:"things"`
	Channels  []things.Channel `toml:"channels" mapstructure:"channels"`
	Cert      Cert             `toml:"cert" mapstructure:"cert"`
}

// Save - store config in a file
func Save(c Config, file string) error {
	b, err := toml.Marshal(c)
	if err != nil {
		return errors.New(fmt.Sprintf("Error reading config file: %s", err))
	}
	if err := ioutil.WriteFile(file, b, 0644); err != nil {
		return errors.New(fmt.Sprintf("Error writing toml: %s", err))
	}
	return nil
}

// Read - retrieve config from a file
func Read(file string) (Config, error) {
	data, err := ioutil.ReadFile(file)
	c := Config{}
	if err != nil {
		return c, errors.New(fmt.Sprintf("Error reading config file: %s", err))
	}

	if err := toml.Unmarshal(data, &c); err != nil {
		return Config{}, errors.New(fmt.Sprintf("Error unmarshaling toml: %s", err))
	}
	return c, nil
}
