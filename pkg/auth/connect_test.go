// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	cases := []struct {
		desc   string
		config Config
		err    error
		secure string
	}{
		{
			desc: "successful without TLS",
			config: Config{
				URL:     "localhost:8080",
				Timeout: time.Second,
			},
			err:    nil,
			secure: "without TLS",
		},
		{
			desc: "successful with TLS",
			config: Config{
				URL:          "localhost:8080",
				Timeout:      time.Second,
				ServerCAFile: "../../docker/ssl/certs/ca.crt",
			},
			err:    nil,
			secure: "with TLS",
		},
		{
			desc: "successful with mTLS",
			config: Config{
				URL:          "localhost:8080",
				Timeout:      time.Second,
				ClientCert:   "../../docker/ssl/certs/magistrala-server.crt",
				ClientKey:    "../../docker/ssl/certs/magistrala-server.key",
				ServerCAFile: "../../docker/ssl/certs/ca.crt",
			},
			err:    nil,
			secure: "with mTLS",
		},
		{
			desc: "failed with empty URL",
			config: Config{
				URL:     "",
				Timeout: time.Second,
			},
			err: errors.New("failed to connect to grpc server"),
		},
		{
			desc: "failed with invalid server CA file",
			config: Config{
				URL:          "localhost:8080",
				Timeout:      time.Second,
				ServerCAFile: "invalid",
			},
			err: errors.New("failed to load root ca file: open invalid: no such file or directory"),
		},
		{
			desc: "failed with invalid server CA file as cert key",
			config: Config{
				URL:          "localhost:8080",
				Timeout:      time.Second,
				ServerCAFile: "../../docker/ssl/certs/magistrala-server.key",
			},
			err: errors.New("failed to append root ca to tls.Config"),
		},
		{
			desc: "failed with invalid client cert",
			config: Config{
				URL:          "localhost:8080",
				Timeout:      time.Second,
				ClientCert:   "invalid",
				ClientKey:    "../../docker/ssl/certs/magistrala-server.key",
				ServerCAFile: "../../docker/ssl/certs/ca.crt",
			},
			err: errors.New("failed to client certificate and key open invalid: no such file or directory"),
		},
		{
			desc: "failed with invalid client key",
			config: Config{
				URL:          "localhost:8080",
				Timeout:      time.Second,
				ClientCert:   "../../docker/ssl/certs/magistrala-server.crt",
				ClientKey:    "invalid",
				ServerCAFile: "../../docker/ssl/certs/ca.crt",
			},
			err: errors.New("failed to client certificate and key open invalid: no such file or directory"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			handler, err := newHandler(c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.Equal(t, c.secure, handler.Secure())
				assert.NotNil(t, handler.Connection())
				assert.Nil(t, handler.Close())
			}
		})
	}
}
