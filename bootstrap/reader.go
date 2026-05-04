// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"io"
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

type reader struct {
	encKey []byte
}

// NewConfigReader return new reader which is used to generate response
// from the config.
func NewConfigReader(encKey []byte) ConfigReader {
	return reader{encKey: encKey}
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
		return r.encrypt(b)
	}

	return res, nil
}

func (r reader) encrypt(in []byte) ([]byte, error) {
	block, err := aes.NewCipher(r.encKey)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(in))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], in)
	return ciphertext, nil
}
