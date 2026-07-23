// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/stretchr/testify/assert"
)

type readResp struct {
	ID         string `json:"id"`
	Content    string `json:"content,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
	CACert     string `json:"ca_cert,omitempty"`
}

func dec(in []byte, cfg bootstrap.Config) ([]byte, error) {
	block, err := aes.NewCipher(cfg.SecureTransportKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, ciphertext := in[:aead.NonceSize()], in[aead.NonceSize():]
	aad := fmt.Sprintf("bootstrap-response:%s:%s:%s:%s", cfg.DomainID, cfg.ExternalID, cfg.SecureTransportKeyID, cfg.SecureRequestID)
	return aead.Open(nil, nonce, ciphertext, []byte(aad))
}

func TestReadConfig(t *testing.T) {
	cfg := bootstrap.Config{
		ID:                   "smq_id",
		DomainID:             "domain-id",
		ExternalID:           "external-id",
		ClientCert:           "client_cert",
		ClientKey:            "client_key",
		CACert:               "ca_cert",
		Content:              "content",
		SecureTransportKey:   encKey,
		SecureTransportKeyID: "key-id",
		SecureRequestID:      "request-id",
	}
	ret := readResp{
		ID:         "smq_id",
		Content:    "content",
		ClientCert: "client_cert",
		ClientKey:  "client_key",
		CACert:     "ca_cert",
	}

	bin, err := json.Marshal(ret)
	assert.Nil(t, err, fmt.Sprintf("Marshalling expected to succeed: %s.\n", err))

	reader := bootstrap.NewConfigReader(encKey)
	cases := []struct {
		desc   string
		config bootstrap.Config
		enc    []byte
		secret bool
		err    error
	}{
		{
			desc:   "read a config",
			config: cfg,
			enc:    bin,
			secret: false,
		},
		{
			desc:   "read encrypted config",
			config: cfg,
			enc:    bin,
			secret: true,
		},
	}

	for _, tc := range cases {
		res, err := reader.ReadConfig(tc.config, tc.secret)
		assert.Nil(t, err, fmt.Sprintf("Reading config to succeed: %s.\n", err))

		if tc.secret {
			encrypted := res.(bootstrap.SecureConfigPayload)
			d, err := dec(encrypted.Payload, tc.config)
			assert.Nil(t, err, fmt.Sprintf("Decrypting expected to succeed: %s.\n", err))
			assert.Equal(t, tc.enc, d, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.enc, d))
			continue
		}
		b, err := json.Marshal(res)
		assert.Nil(t, err, fmt.Sprintf("Marshalling expected to succeed: %s.\n", err))
		assert.Equal(t, tc.enc, b, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.enc, b))
		resp, ok := res.(magistrala.Response)
		assert.True(t, ok, "If not encrypted, reader should return response.")
		assert.False(t, resp.Empty(), fmt.Sprintf("Response should not be empty %s.", err))
		assert.Equal(t, http.StatusOK, resp.Code(), "Default config response code should be 200.")
	}
}
