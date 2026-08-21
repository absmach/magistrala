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
	ID          string      `json:"id,omitempty"`
	ContentType ContentType `json:"content_type"`
	Content     string      `json:"content"`
	ClientCert  string      `json:"client_cert,omitempty"`
	ClientKey   string      `json:"client_key,omitempty"`
	CACert      string      `json:"ca_cert,omitempty"`
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

func (res EncryptedBootstrapConfig) Code() int {
	return http.StatusOK
}

func (res EncryptedBootstrapConfig) Headers() map[string]string {
	return map[string]string{"Cache-Control": "no-store"}
}

func (res EncryptedBootstrapConfig) Empty() bool {
	return false
}

// NewConfigReader return new reader which is used to generate response
// from the config.
func NewConfigReader() ConfigReader {
	return reader{}
}

func (r reader) ReadConfig(cfg Config) (any, error) {
	contentType := cfg.ContentType
	if contentType == "" {
		contentType = ContentTypeTextPlain
	}
	res := bootstrapRes{
		ID:          cfg.ID,
		ContentType: contentType,
		Content:     cfg.Content,
		ClientCert:  cfg.ClientCert,
		ClientKey:   cfg.ClientKey,
		CACert:      cfg.CACert,
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	return encryptDeviceBootstrapResponse(cfg, b)
}
