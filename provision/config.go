// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"
	"os"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/pelletier/go-toml"
)

var errFailedToReadConfig = errors.New("failed to read config file")

// ServiceConf represents service config.
type ServiceConf struct {
	Port           string `toml:"port"          env:"MG_PROVISION_HTTP_PORT"            envDefault:"9016"`
	LogLevel       string `toml:"log_level"     env:"MG_PROVISION_LOG_LEVEL"            envDefault:"info"`
	TLS            bool   `toml:"tls"           env:"MG_PROVISION_ENV_CLIENTS_TLS"      envDefault:"false"`
	ServerCert     string `toml:"server_cert"   env:"MG_PROVISION_SERVER_CERT"          envDefault:""`
	ServerKey      string `toml:"server_key"    env:"MG_PROVISION_SERVER_KEY"           envDefault:""`
	ThingsURL      string `toml:"things_url"    env:"MG_PROVISION_THINGS_LOCATION"      envDefault:"http://localhost"`
	UsersURL       string `toml:"users_url"     env:"MG_PROVISION_USERS_LOCATION"       envDefault:"http://localhost"`
	HTTPPort       string `toml:"http_port"     env:"MG_PROVISION_HTTP_PORT"            envDefault:"9016"`
	MgUser         string `toml:"mg_user"       env:"MG_PROVISION_USER"                 envDefault:"test@example.com"`
	MgPass         string `toml:"mg_pass"       env:"MG_PROVISION_PASS"                 envDefault:"test"`
	MgDomainID     string `toml:"mg_domain_id"  env:"MG_PROVISION_DOMAIN_ID"            envDefault:""`
	MgAPIKey       string `toml:"mg_api_key"    env:"MG_PROVISION_API_KEY"              envDefault:""`
	MgBSURL        string `toml:"mg_bs_url"     env:"MG_PROVISION_BS_SVC_URL"           envDefault:"http://localhost:9000/things/configs"`
	MgWhiteListURL string `toml:"mg_white_list" env:"MG_PROVISION_BS_SVC_WHITELIST_URL" envDefault:"http://localhost:9000/things/state"`
	MgCertsURL     string `toml:"mg_certs_url"  env:"MG_PROVISION_CERTS_SVC_URL"        envDefault:"http://localhost:9019"`
}

// Bootstrap represetns the Bootstrap config.
type Bootstrap struct {
	X509Provision bool                   `toml:"x509_provision" env:"MG_PROVISION_X509_PROVISIONING"      envDefault:"false"`
	Provision     bool                   `toml:"provision"      env:"MG_PROVISION_BS_CONFIG_PROVISIONING" envDefault:"true"`
	AutoWhiteList bool                   `toml:"autowhite_list" env:"MG_PROVISION_BS_AUTO_WHITELIST"      envDefault:"true"`
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
	TTL string `json:"ttl" toml:"ttl" env:"MG_PROVISION_CERTS_HOURS_VALID" envDefault:"2400h"`
}

// Config struct of Provision.
type Config struct {
	LogLevel      string             `toml:"log_level" env:"MG_PROVISION_LOG_LEVEL" envDefault:"info"`
	File          string             `toml:"file"      env:"MG_PROVISION_CONFIG_FILE" envDefault:"config.toml"`
	Server        ServiceConf        `toml:"server"    mapstructure:"server"`
	Bootstrap     Bootstrap          `toml:"bootstrap" mapstructure:"bootstrap"`
	Things        []mgclients.Client `toml:"things"    mapstructure:"things"`
	Channels      []groups.Group     `toml:"channels"  mapstructure:"channels"`
	Cert          Cert               `toml:"cert"      mapstructure:"cert"`
	BSContent     string             `env:"MG_PROVISION_BS_CONTENT" envDefault:""`
	SendTelemetry bool               `env:"MG_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID    string             `env:"MG_MQTT_ADAPTER_INSTANCE_ID" envDefault:""`
}

// Save - store config in a file.
func Save(c Config, file string) error {
	if file == "" {
		return errors.ErrEmptyPath
	}

	b, err := toml.Marshal(c)
	if err != nil {
		return errors.Wrap(errFailedToReadConfig, err)
	}
	if err := os.WriteFile(file, b, 0o644); err != nil {
		return fmt.Errorf("Error writing toml: %w", err)
	}

	return nil
}

// Read - retrieve config from a file.
func Read(file string) (Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return Config{}, errors.Wrap(errFailedToReadConfig, err)
	}

	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("Error unmarshaling toml: %w", err)
	}

	return c, nil
}
