// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"io"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/absmach/magistrala/pkg/errors"
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/pelletier/go-toml"
	"github.com/spf13/cobra"
)

type remotes struct {
	ThingsURL       string `toml:"things_url"`
	UsersURL        string `toml:"users_url"`
	ReaderURL       string `toml:"reader_url"`
	HTTPAdapterURL  string `toml:"http_adapter_url"`
	BootstrapURL    string `toml:"bootstrap_url"`
	CertsURL        string `toml:"certs_url"`
	TLSVerification bool   `toml:"tls_verification"`
}

type filter struct {
	Offset string `toml:"offset"`
	Limit  string `toml:"limit"`
	Topic  string `toml:"topic"`
}

type config struct {
	Remotes   remotes `toml:"remotes"`
	Filter    filter  `toml:"filter"`
	UserToken string  `toml:"user_token"`
	RawOutput string  `toml:"raw_output"`
}

// Readable by all user groups but writeable by the user only.
const filePermission = 0o644

var (
	errReadFail            = errors.New("failed to read config file")
	errNoKey               = errors.New("no such key")
	errUnsupportedKeyValue = errors.New("unsupported data type for key")
	errWritingConfig       = errors.New("error in writing the updated config to file")
	errInvalidURL          = errors.New("invalid url")
	errURLParseFail        = errors.New("failed to parse url")
	defaultConfigPath      = "./config.toml"
)

func read(file string) (config, error) {
	c := config{}
	data, err := os.Open(file)
	if err != nil {
		return c, errors.Wrap(errReadFail, err)
	}
	defer data.Close()

	buf, err := io.ReadAll(data)
	if err != nil {
		return c, errors.Wrap(errReadFail, err)
	}

	if err := toml.Unmarshal(buf, &c); err != nil {
		return config{}, err
	}

	return c, nil
}

// ParseConfig - parses the config file.
func ParseConfig(sdkConf mgxsdk.Config) (mgxsdk.Config, error) {
	if ConfigPath == "" {
		ConfigPath = defaultConfigPath
	}

	_, err := os.Stat(ConfigPath)
	switch {
	// If the file does not exist, create it with default values.
	case os.IsNotExist(err):
		defaultConfig := config{
			Remotes: remotes{
				ThingsURL:       "http://localhost:9000",
				UsersURL:        "http://localhost:9002",
				ReaderURL:       "http://localhost",
				HTTPAdapterURL:  "http://localhost/http:9016",
				BootstrapURL:    "http://localhost",
				CertsURL:        "https://localhost:9019",
				TLSVerification: false,
			},
		}
		buf, err := toml.Marshal(defaultConfig)
		if err != nil {
			return sdkConf, err
		}
		if err = os.WriteFile(ConfigPath, buf, filePermission); err != nil {
			return sdkConf, errors.Wrap(errWritingConfig, err)
		}
	case err != nil:
		return sdkConf, err
	}

	config, err := read(ConfigPath)
	if err != nil {
		return sdkConf, err
	}

	if config.Filter.Offset != "" {
		offset, err := strconv.ParseUint(config.Filter.Offset, 10, 64)
		if err != nil {
			return sdkConf, err
		}
		Offset = offset
	}

	if config.Filter.Limit != "" {
		limit, err := strconv.ParseUint(config.Filter.Limit, 10, 64)
		if err != nil {
			return sdkConf, err
		}
		Limit = limit
	}

	if config.Filter.Topic != "" {
		Topic = config.Filter.Topic
	}

	if config.RawOutput != "" {
		rawOutput, err := strconv.ParseBool(config.RawOutput)
		if err != nil {
			return sdkConf, err
		}
		RawOutput = rawOutput
	}

	sdkConf.ThingsURL = config.Remotes.ThingsURL
	sdkConf.UsersURL = config.Remotes.UsersURL
	sdkConf.ReaderURL = config.Remotes.ReaderURL
	sdkConf.HTTPAdapterURL = config.Remotes.HTTPAdapterURL
	sdkConf.BootstrapURL = config.Remotes.BootstrapURL
	sdkConf.CertsURL = config.Remotes.CertsURL

	return sdkConf, nil
}

// New config command to store params to local TOML file.
func NewConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config <key> <value>",
		Short: "CLI local config",
		Long:  "Local param storage to prevent repetitive passing of keys",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := setConfigValue(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	}
}

func setConfigValue(key, value string) error {
	config, err := read(ConfigPath)
	if err != nil {
		return err
	}

	if strings.Contains(key, "url") {
		u, err := url.Parse(value)
		if err != nil {
			return errors.Wrap(errInvalidURL, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return errors.Wrap(errInvalidURL, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.Wrap(errURLParseFail, err)
		}
	}

	configKeyToField := map[string]interface{}{
		"things_url":       &config.Remotes.ThingsURL,
		"users_url":        &config.Remotes.UsersURL,
		"reader_url":       &config.Remotes.ReaderURL,
		"http_adapter_url": &config.Remotes.HTTPAdapterURL,
		"bootstrap_url":    &config.Remotes.BootstrapURL,
		"certs_url":        &config.Remotes.CertsURL,
		"tls_verification": &config.Remotes.TLSVerification,
		"offset":           &config.Filter.Offset,
		"limit":            &config.Filter.Limit,
		"topic":            &config.Filter.Topic,
		"raw_output":       &config.RawOutput,
		"user_token":       &config.UserToken,
	}

	fieldPtr, ok := configKeyToField[key]
	if !ok {
		return errNoKey
	}

	fieldValue := reflect.ValueOf(fieldPtr).Elem()

	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)
	case reflect.Int:
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		fieldValue.SetUint(uint64(intValue))
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		fieldValue.SetBool(boolValue)
	default:
		return errors.Wrap(errUnsupportedKeyValue, err)
	}

	buf, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	if err = os.WriteFile(ConfigPath, buf, filePermission); err != nil {
		return errors.Wrap(errWritingConfig, err)
	}

	return nil
}
