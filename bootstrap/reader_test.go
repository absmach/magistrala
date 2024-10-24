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
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type readChan struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type readResp struct {
	ClientID     string     `json:"client_id"`
	ClientSecret string     `json:"client_secret"`
	Channels     []readChan `json:"channels"`
	Content      string     `json:"content,omitempty"`
	ClientCert   string     `json:"client_cert,omitempty"`
	ClientKey    string     `json:"client_key,omitempty"`
	CACert       string     `json:"ca_cert,omitempty"`
}

func dec(in []byte) ([]byte, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	if len(in) < aes.BlockSize {
		return nil, errors.ErrMalformedEntity
	}
	iv := in[:aes.BlockSize]
	in = in[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(in, in)
	return in, nil
}

func TestReadConfig(t *testing.T) {
	cfg := bootstrap.Config{
		ClientID:     "mg_id",
		ClientCert:   "client_cert",
		ClientKey:    "client_key",
		CACert:       "ca_cert",
		ClientSecret: "mg_key",
		Channels: []bootstrap.Channel{
			{
				ID:       "mg_id",
				Name:     "mg_name",
				Metadata: map[string]interface{}{"key": "value}"},
			},
		},
		Content: "content",
	}
	ret := readResp{
		ClientID:     "mg_id",
		ClientSecret: "mg_key",
		Channels: []readChan{
			{
				ID:       "mg_id",
				Name:     "mg_name",
				Metadata: map[string]interface{}{"key": "value}"},
			},
		},
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
			d, err := dec(res.([]byte))
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
