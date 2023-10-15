// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"
	"os"

	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/groups"
	"github.com/pelletier/go-toml"
)

// ServiceConf represents service config.
type ServiceConf struct {
	Port           string `toml:"port"          env:"MF_PROVISION_HTTP_PORT"            envDefault:"9016"`
	LogLevel       string `toml:"log_level"     env:"MF_PROVISION_LOG_LEVEL"            envDefault:"info"`
	TLS            bool   `toml:"tls"           env:"MF_PROVISION_ENV_CLIENTS_TLS"      envDefault:"false"`
	ServerCert     string `toml:"server_cert"   env:"MF_PROVISION_SERVER_CERT"          envDefault:""`
	ServerKey      string `toml:"server_key"    env:"MF_PROVISION_SERVER_KEY"           envDefault:""`
	ThingsURL      string `toml:"things_url"    env:"MF_PROVISION_THINGS_LOCATION"      envDefault:"http://localhost"`
	UsersURL       string `toml:"users_url"     env:"MF_PROVISION_USERS_LOCATION"       envDefault:"http://localhost"`
	HTTPPort       string `toml:"http_port"     env:"MF_PROVISION_HTTP_PORT"            envDefault:"9016"`
	MfUser         string `toml:"mf_user"       env:"MF_PROVISION_USER"                 envDefault:"test@example.com"`
	MfPass         string `toml:"mf_pass"       env:"MF_PROVISION_PASS"                 envDefault:"test"`
	MfAPIKey       string `toml:"mf_api_key"    env:"MF_PROVISION_API_KEY"              envDefault:""`
	MfBSURL        string `toml:"mf_bs_url"     env:"MF_PROVISION_BS_SVC_URL"           envDefault:"http://localhost:9000/things/configs"`
	MfWhiteListURL string `toml:"mf_white_list" env:"MF_PROVISION_BS_SVC_WHITELIST_URL" envDefault:"http://localhost:9000/things/state"`
	MfCertsURL     string `toml:"mf_certs_url"  env:"MF_PROVISION_CERTS_SVC_URL"        envDefault:"http://localhost:9019"`
}

// Bootstrap represetns the Bootstrap config.
type Bootstrap struct {
	X509Provision bool                   `toml:"x509_provision" env:"MF_PROVISION_X509_PROVISIONING"      envDefault:"false"`
	Provision     bool                   `toml:"provision"      env:"MF_PROVISION_BS_CONFIG_PROVISIONING" envDefault:"true"`
	AutoWhiteList bool                   `toml:"autowhite_list" env:"MF_PROVISION_BS_AUTO_WHITELIST"      envDefault:"true"`
	Content       map[string]interface{} `toml:"content"`
}

// Gateway represetns the Gateway config.
type Gateway struct {
	Type            string `toml:"type" json:"type"`
	ExternalID      string `toml:"external_id" json:"external_id"`
	ExternalKey     string `toml:"external_key" json:"external_key"`
	CtrlChannelID   string `toml:"ctrl_channel_id" json:"ctrl_channel_id"`
	DataChannelID   string `toml:"data_channel_id" json:"data_channel_id"`
	ExportChannelID string `toml:"export_channel_id" json:"export_channel_id"`
	CfgID           string `toml:"cfg_id" json:"cfg_id"`
}

// Cert represetns the certificate config.
type Cert struct {
	TTL string `json:"ttl" toml:"ttl" env:"MF_PROVISION_CERTS_HOURS_VALID" envDefault:"2400h"`
}

// Config struct of Provision.
type Config struct {
	LogLevel      string             `toml:"log_level" env:"MF_PROVISION_LOG_LEVEL" envDefault:"info"`
	File          string             `toml:"file"      env:"MF_PROVISION_CONFIG_FILE" envDefault:"config.toml"`
	Server        ServiceConf        `toml:"server"    mapstructure:"server"`
	Bootstrap     Bootstrap          `toml:"bootstrap" mapstructure:"bootstrap"`
	Things        []mfclients.Client `toml:"things"    mapstructure:"things"`
	Channels      []groups.Group     `toml:"channels"  mapstructure:"channels"`
	Cert          Cert               `toml:"cert"      mapstructure:"cert"`
	BSContent     string             `env:"MF_PROVISION_BS_CONTENT" envDefault:""`
	SendTelemetry bool               `env:"MF_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID    string             `env:"MF_MQTT_ADAPTER_INSTANCE_ID" envDefault:""`
}

// Save - store config in a file.
func Save(c Config, file string) error {
	b, err := toml.Marshal(c)
	if err != nil {
		return errors.New(fmt.Sprintf("Error reading config file: %s", err))
	}
	if err := os.WriteFile(file, b, 0o644); err != nil {
		return errors.New(fmt.Sprintf("Error writing toml: %s", err))
	}
	return nil
}

// Read - retrieve config from a file.
func Read(file string) (Config, error) {
	data, err := os.ReadFile(file)
	c := Config{}
	if err != nil {
		return c, errors.New(fmt.Sprintf("Error reading config file: %s", err))
	}

	if err := toml.Unmarshal(data, &c); err != nil {
		return Config{}, errors.New(fmt.Sprintf("Error unmarshaling toml: %s", err))
	}
	return c, nil
}
