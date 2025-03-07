// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs_test

import (
	"fmt"
	"testing"

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestLoadCertificates(t *testing.T) {
	cases := []struct {
		desc      string
		caPath    string
		caKeyPath string
		err       error
	}{
		{
			desc:      "load valid tls certificate and valid key",
			caPath:    "../docker/ssl/certs/ca.crt",
			caKeyPath: "../docker/ssl/certs/ca.key",
			err:       nil,
		},
		{
			desc:      "load valid tls certificate and missing key",
			caPath:    "../docker/ssl/certs/ca.crt",
			caKeyPath: "",
			err:       certs.ErrMissingCerts,
		},
		{
			desc:      "load missing tls certificate and valid key",
			caPath:    "",
			caKeyPath: "../docker/ssl/certs/ca.key",
			err:       certs.ErrMissingCerts,
		},
		{
			desc:      "load empty tls certificate and empty key",
			caPath:    "",
			caKeyPath: "",
			err:       certs.ErrMissingCerts,
		},
		{
			desc:      "load valid tls certificate and invalid key",
			caPath:    "../docker/ssl/certs/ca.crt",
			caKeyPath: "certs.go",
			err:       errors.New("tls: failed to find any PEM data in key input"),
		},
		{
			desc:      "load invalid tls certificate and valid key",
			caPath:    "certs.go",
			caKeyPath: "../docker/ssl/certs/ca.key",
			err:       errors.New("tls: failed to find any PEM data in certificate input"),
		},
		{
			desc:      "load invalid tls certificate and invalid key",
			caPath:    "certs.go",
			caKeyPath: "certs.go",
			err:       errors.New("tls: failed to find any PEM data in certificate input"),
		},

		{
			desc:      "load valid tls certificate and non-existing key",
			caPath:    "../docker/ssl/certs/ca.crt",
			caKeyPath: "ca.key",
			err:       errors.New("stat ca.key: no such file or directory"),
		},
		{
			desc:      "load non-existing tls certificate and valid key",
			caPath:    "ca.crt",
			caKeyPath: "../docker/ssl/certs/ca.key",
			err:       errors.New("stat ca.crt: no such file or directory"),
		},
		{
			desc:      "load non-existing tls certificate and non-existing key",
			caPath:    "ca.crt",
			caKeyPath: "ca.key",
			err:       errors.New("stat ca.crt: no such file or directory"),
		},
	}

	for _, tc := range cases {
		tlsCert, caCert, err := certs.LoadCertificates(tc.caPath, tc.caKeyPath)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotNil(t, tlsCert)
			assert.NotNil(t, caCert)
		}
	}
}
