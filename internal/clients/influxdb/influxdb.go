package influxdb

import (
	"fmt"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errConnect = errors.New("failed to create InfluxDB client")
	errConfig  = errors.New("failed to load InfluxDB client configuration from environment variable")
)

type Config struct {
	Protocol           string        `env:"PROTOCOL"              envDefault:"http"`
	Host               string        `env:"HOST"                  envDefault:"localhost"`
	Port               string        `env:"PORT"                  envDefault:"8086"`
	Username           string        `env:"ADMIN_USER"            envDefault:"mainflux"`
	Password           string        `env:"ADMIN_PASSWORD"        envDefault:"mainflux"`
	DbName             string        `env:"DB"                    envDefault:"mainflux"`
	UserAgent          string        `env:"USER_AGENT"            envDefault:"InfluxDBClient"`
	Timeout            time.Duration `env:"TIMEOUT"` // Influxdb client configuration by default there is no timeout duration , this field will not have fallback default timeout duration Reference: https://pkg.go.dev/github.com/influxdata/influxdb@v1.10.0/client/v2#HTTPConfig
	InsecureSkipVerify bool          `env:"INSECURE_SKIP_VERIFY"  envDefault:"false"`
}

// Setup load configuration from environment variable, create InfluxDB client and connect to InfluxDB server
func Setup(envPrefix string) (client.HTTPClient, error) {
	config := Config{}
	if err := env.Parse(&config, env.Options{Prefix: envPrefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	return Connect(config)
}

// Connect create InfluxDB client and connect to InfluxDB server
func Connect(config Config) (client.HTTPClient, error) {
	address := fmt.Sprintf("%s://%s:%s", config.Protocol, config.Host, config.Port)
	clientConfig := client.HTTPConfig{
		Addr:      address,
		Username:  config.Username,
		Password:  config.Password,
		UserAgent: config.UserAgent,
		Timeout:   config.Timeout,
	}
	client, err := client.NewHTTPClient(clientConfig)
	if err != nil {
		return nil, errors.Wrap(errConnect, err)
	}
	return client, nil
}
