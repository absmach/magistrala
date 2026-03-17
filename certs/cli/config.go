// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"io"
	"os"
	"strconv"

	ctxsdk "github.com/absmach/supermq/certs/sdk"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/pelletier/go-toml"
)

const (
	defURL             string = "http://localhost"
	defCertsURL        string = defURL + ":9010"
	defTLSVerification bool   = false
	defOffset          string = "0"
	defLimit           string = "10"
	defTopic           string = ""
	defRawOutput       string = "false"
)

type remotes struct {
	CertsURL        string `toml:"certs_url"`
	HostURL         string `toml:"host_url"`
	TLSVerification bool   `toml:"tls_verification"`
}

type filter struct {
	Offset string `toml:"offset"`
	Limit  string `toml:"limit"`
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
	errReadFail       = errors.New("failed to read config file")
	errWritingConfig  = errors.New("error in writing the updated config to file")
	defaultConfigPath = "./config.toml"
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
func ParseConfig(sdkConf ctxsdk.Config) (ctxsdk.Config, error) {
	if ConfigPath == "" {
		ConfigPath = defaultConfigPath
	}

	_, err := os.Stat(ConfigPath)
	switch {
	// If the file does not exist, create it with default values.
	case os.IsNotExist(err):
		defaultConfig := config{
			Remotes: remotes{
				CertsURL:        defCertsURL,
				HostURL:         defURL,
				TLSVerification: defTLSVerification,
			},
			Filter: filter{
				Offset: defOffset,
				Limit:  defLimit,
			},
			RawOutput: defRawOutput,
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

	if config.Filter.Offset != "" && Offset == 0 {
		offset, err := strconv.ParseUint(config.Filter.Offset, 10, 64)
		if err != nil {
			return sdkConf, err
		}
		Offset = offset
	}

	if config.Filter.Limit != "" && Limit == 0 {
		limit, err := strconv.ParseUint(config.Filter.Limit, 10, 64)
		if err != nil {
			return sdkConf, err
		}
		Limit = limit
	}

	if config.RawOutput != "" {
		rawOutput, err := strconv.ParseBool(config.RawOutput)
		if err != nil {
			return sdkConf, err
		}
		// check for config file value or flag input value is true
		RawOutput = rawOutput || RawOutput
	}

	if sdkConf.CertsURL == "" && config.Remotes.CertsURL != "" {
		sdkConf.CertsURL = config.Remotes.CertsURL
	}

	if sdkConf.HostURL == "" && config.Remotes.HostURL != "" {
		sdkConf.HostURL = config.Remotes.HostURL
	}

	sdkConf.TLSVerification = config.Remotes.TLSVerification || sdkConf.TLSVerification

	return sdkConf, nil
}
