// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package provision_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/provision"
	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
)

var (
	validConfig = provision.Config{
		Server: provision.ServiceConf{
			Port:     "9016",
			LogLevel: "info",
			TLS:      false,
		},
		Bootstrap: provision.Bootstrap{
			X509Provision: true,
			Provision:     true,
			AutoWhiteList: true,
			Content: map[string]interface{}{
				"test": "test",
			},
		},
		Clients: []clients.Client{
			{
				ID:   "1234567890",
				Name: "test",
				Tags: []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				Permissions: []string{"test"},
			},
		},
		Channels: []channels.Channel{
			{
				ID:   "1234567890",
				Name: "test",
				Metadata: map[string]interface{}{
					"test": "test",
				},
				Permissions: []string{"test"},
			},
		},
		Cert:          provision.Cert{},
		SendTelemetry: true,
		InstanceID:    "1234567890",
	}
	validConfigFile = "./config.toml"
	invalidConfig   = provision.Config{
		Bootstrap: provision.Bootstrap{
			Content: map[string]interface{}{
				"invalid": make(chan int),
			},
		},
	}
	invalidConfigFile = "./invalid.toml"
)

func createInvalidConfigFile() error {
	config := map[string]interface{}{
		"invalid": "invalid",
	}
	b, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	f, err := os.Create(invalidConfigFile)
	if err != nil {
		return err
	}

	if _, err = f.Write(b); err != nil {
		return err
	}

	return nil
}

func createValidConfigFile() error {
	b, err := toml.Marshal(validConfig)
	if err != nil {
		return err
	}

	f, err := os.Create(validConfigFile)
	if err != nil {
		return err
	}

	if _, err = f.Write(b); err != nil {
		return err
	}

	return nil
}

func TestSave(t *testing.T) {
	cases := []struct {
		desc string
		cfg  provision.Config
		file string
		err  error
	}{
		{
			desc: "save valid config",
			cfg:  validConfig,
			file: validConfigFile,
			err:  nil,
		},
		{
			desc: "save valid config with empty file name",
			cfg:  validConfig,
			file: "",
			err:  errors.ErrEmptyPath,
		},
		{
			desc: "save empty config with valid config file",
			cfg:  provision.Config{},
			file: validConfigFile,
			err:  nil,
		},
		{
			desc: "save empty config with empty file name",
			cfg:  provision.Config{},
			file: "",
			err:  errors.ErrEmptyPath,
		},
		{
			desc: "save invalid config",
			cfg:  invalidConfig,
			file: invalidConfigFile,
			err:  errors.New("failed to read config file"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := provision.Save(c.cfg, c.file)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected: %v, got: %v", c.err, err))

			if err == nil {
				defer func() {
					if c.file != "" {
						err := os.Remove(c.file)
						assert.NoError(t, err)
					}
				}()

				cfg, err := provision.Read(c.file)
				if c.cfg.Bootstrap.Content == nil {
					c.cfg.Bootstrap.Content = map[string]interface{}{}
				}
				assert.Equal(t, c.err, err)
				assert.Equal(t, c.cfg, cfg)
			}
		})
	}
}

func TestRead(t *testing.T) {
	err := createInvalidConfigFile()
	assert.NoError(t, err)

	err = createValidConfigFile()
	assert.NoError(t, err)

	t.Cleanup(func() {
		err := os.Remove(invalidConfigFile)
		assert.NoError(t, err)
		err = os.Remove(validConfigFile)
		assert.NoError(t, err)
	})

	cases := []struct {
		desc string
		file string
		cfg  provision.Config
		err  error
	}{
		{
			desc: "read valid config",
			file: validConfigFile,
			cfg:  validConfig,
			err:  nil,
		},
		{
			desc: "read invalid config",
			file: invalidConfigFile,
			cfg:  invalidConfig,
			err:  nil,
		},
		{
			desc: "read empty config",
			file: "",
			cfg:  provision.Config{},
			err:  errors.New("failed to read config file"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cfg, err := provision.Read(c.file)
			if c.desc == "read invalid config" {
				c.cfg.Bootstrap.Content = nil
			}
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected: %v, got: %v", c.err, err))
			assert.Equal(t, c.cfg, cfg)
		})
	}
}
