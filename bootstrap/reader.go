// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"encoding/json"
	"net/http"
)

// bootstrapRes represent Magistrala Response to the Bootstrap request.
// This is used as a response from ConfigReader and can easily be
// replaced with any other response format.
type bootstrapRes struct {
	ID         string `json:"id,omitempty"`
	Content    string `json:"content,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
	CACert     string `json:"ca_cert,omitempty"`
}

func (res bootstrapRes) Code() int {
	return http.StatusOK
}

func (res bootstrapRes) Headers() map[string]string {
	return map[string]string{}
}

func (res bootstrapRes) Empty() bool {
	return false
}

type reader struct{}

// SecureConfigPayload carries the encrypted response and non-secret envelope
// metadata to the HTTP transport.
type SecureConfigPayload struct {
	Payload   []byte
	KeyID     string
	RequestID string
}

// NewConfigReader return new reader which is used to generate response
// from the config.
func NewConfigReader(_ []byte) ConfigReader {
	return reader{}
}

func (r reader) ReadConfig(cfg Config, secure bool) (any, error) {
	res := bootstrapRes{
		ID:         cfg.ID,
		Content:    cfg.Content,
		ClientCert: cfg.ClientCert,
		ClientKey:  cfg.ClientKey,
		CACert:     cfg.CACert,
	}
	if secure {
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}
		payload, err := encryptSecureResponse(cfg, b)
		if err != nil {
			return nil, err
		}
		return SecureConfigPayload{Payload: payload, KeyID: cfg.SecureTransportKeyID, RequestID: cfg.SecureRequestID}, nil
	}

	return res, nil
}
