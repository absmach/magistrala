// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestSetupAuth(t *testing.T) {
	cases := []struct {
		desc   string
		config auth.Config
		err    error
	}{
		{
			desc: "successful",
			config: auth.Config{
				URL:     "localhost:8080",
				Timeout: time.Second,
			},
			err: nil,
		},
		{
			desc: "failed with empty URL",
			config: auth.Config{
				URL:     "",
				Timeout: time.Second,
			},
			err: errors.New("failed to connect to grpc server"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			client, handler, err := auth.Setup(c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}

func TestSetupAuthz(t *testing.T) {
	cases := []struct {
		desc   string
		config auth.Config
		err    error
	}{
		{
			desc: "successful",
			config: auth.Config{
				URL:     "localhost:8080",
				Timeout: time.Second,
			},
			err: nil,
		},
		{
			desc: "failed with empty URL",
			config: auth.Config{
				URL:     "",
				Timeout: time.Second,
			},
			err: errors.New("failed to connect to grpc server"),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			client, handler, err := auth.SetupAuthz(c.config)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s", err, c.err))
			if err == nil {
				assert.NotNil(t, client)
				assert.NotNil(t, handler)
			}
		})
	}
}
