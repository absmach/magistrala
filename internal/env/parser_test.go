// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/clients/grpc"
	"github.com/absmach/magistrala/internal/server"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var errNotDuration error = errors.New("unable to parse duration")

func TestParseServerConfig(t *testing.T) {
	tests := []struct {
		description    string
		config         *server.Config
		expectedConfig *server.Config
		options        []Options
		err            error
	}{
		{
			"Parsing with Server Config",
			&server.Config{},
			&server.Config{
				Host:         "localhost",
				Port:         "8080",
				CertFile:     "cert",
				KeyFile:      "key",
				ServerCAFile: "server-ca-certs",
				ClientCAFile: "client-ca-certs",
			},
			[]Options{
				{
					Environment: map[string]string{
						"HOST":            "localhost",
						"PORT":            "8080",
						"SERVER_CERT":     "cert",
						"SERVER_KEY":      "key",
						"SERVER_CA_CERTS": "server-ca-certs",
						"CLIENT_CA_CERTS": "client-ca-certs",
					},
				},
			},
			nil,
		},
		{
			"Parsing with Server Config with Prefix",
			&server.Config{},
			&server.Config{
				Host:         "localhost",
				Port:         "8080",
				CertFile:     "cert",
				KeyFile:      "key",
				ServerCAFile: "server-ca-certs",
				ClientCAFile: "client-ca-certs",
			},
			[]Options{
				{
					Environment: map[string]string{
						"MG-HOST":            "localhost",
						"MG-PORT":            "8080",
						"MG-SERVER_CERT":     "cert",
						"MG-SERVER_KEY":      "key",
						"MG-SERVER_CA_CERTS": "server-ca-certs",
						"MG-CLIENT_CA_CERTS": "client-ca-certs",
					},
					Prefix: "MG-",
				},
			},
			nil,
		},
	}
	for _, test := range tests {
		err := Parse(test.config, test.options...)
		switch test.err {
		case nil:
			assert.NoError(t, err, fmt.Sprintf("%s: expected no error but got %v", test.description, err))
		default:
			assert.Error(t, err, fmt.Sprintf("%s: expected error but got nil", test.description))
		}
		assert.Equal(t, test.expectedConfig, test.config, fmt.Sprintf("%s: expected %v got %v", test.description, test.expectedConfig, test.config))
	}
}

func TestParseGRPCConfig(t *testing.T) {
	tests := []struct {
		description    string
		config         *grpc.Config
		expectedConfig *grpc.Config
		options        []Options
		err            error
	}{
		{
			"Parsing a grpc.Config struct",
			&grpc.Config{},
			&grpc.Config{
				URL:     "val.com",
				Timeout: time.Second,
			},
			[]Options{
				{
					Environment: map[string]string{
						"URL":     "val.com",
						"TIMEOUT": time.Second.String(),
					},
				},
			},
			nil,
		},
		{
			"Invalid type parsing",
			&grpc.Config{},
			&grpc.Config{URL: "val.com"},
			[]Options{
				{
					Environment: map[string]string{
						"URL":     "val.com",
						"TIMEOUT": "invalid",
					},
				},
			},
			errNotDuration,
		},
		{
			"Parsing all configs",
			&grpc.Config{},
			&grpc.Config{
				URL:          "val.com",
				Timeout:      time.Second,
				ServerCAFile: "server-ca-cert",
				ClientCert:   "client-cert",
				ClientKey:    "client-key",
			},
			[]Options{
				{
					Environment: map[string]string{
						"MG-URL":             "val.com",
						"MG-TIMEOUT":         "1s",
						"MG-SERVER_CA_CERTS": "server-ca-cert",
						"MG-CLIENT_CERT":     "client-cert",
						"MG-CLIENT_KEY":      "client-key",
					},
					Prefix: "MG-",
				},
			},
			nil,
		},
	}
	for _, test := range tests {
		err := Parse(test.config, test.options...)
		switch test.err {
		case nil:
			assert.NoError(t, err, fmt.Sprintf("%s: expected no error but got %v", test.description, err))
		default:
			assert.Error(t, err, fmt.Sprintf("%s: expected error but got nil", test.description))
		}
		assert.Equal(t, test.expectedConfig, test.config, fmt.Sprintf("%s: expected %v got %v", test.description, test.expectedConfig, test.config))
	}
}

func TestParseCustomConfig(t *testing.T) {
	type CustomConfig struct {
		Field1 string `env:"FIELD1" envDefault:"val1"`
		Field2 int    `env:"FIELD2"`
	}

	tests := []struct {
		description    string
		config         *CustomConfig
		expectedConfig *CustomConfig
		options        []Options
		err            error
	}{
		{
			"parse with missing required field",
			&CustomConfig{},
			&CustomConfig{Field1: "test val"},
			[]Options{
				{
					Environment: map[string]string{
						"FIELD1": "test val",
					},
					RequiredIfNoDef: true,
				},
			},
			errors.New(`required environment variable "FIELD2" not set`),
		},
		{
			"parse with wrong type",
			&CustomConfig{},
			&CustomConfig{Field1: "test val"},
			[]Options{
				{
					Environment: map[string]string{
						"FIELD1": "test val",
						"FIELD2": "not int",
					},
				},
			},
			errors.New(`strconv.ParseInt`),
		},
		{
			"parse with prefix",
			&CustomConfig{},
			&CustomConfig{Field1: "test val", Field2: 2},
			[]Options{
				{
					Environment: map[string]string{
						"MG-FIELD1": "test val",
						"MG-FIELD2": "2",
					},
					Prefix: "MG-",
				},
			},
			nil,
		},
	}

	for _, test := range tests {
		err := Parse(test.config, test.options...)
		switch test.err {
		case nil:
			assert.NoError(t, err, fmt.Sprintf("expected no error but got %v", err))
		default:
			assert.Error(t, err, "expected error but got nil")
		}
		assert.Equal(t, test.expectedConfig, test.config, fmt.Sprintf("expected %v got %v", test.expectedConfig, test.config))
	}
}
