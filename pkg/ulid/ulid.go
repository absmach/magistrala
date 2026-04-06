// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package ulid provides a ULID identity provider.
package ulid

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/oklog/ulid/v2"
)

// ErrGeneratingID indicates error in generating ULID.
var ErrGeneratingID = errors.New("generating id failed")

var _ magistrala.IDProvider = (*ulidProvider)(nil)

type ulidProvider struct {
	entropy io.Reader
}

// New instantiates a ULID provider.
func New() magistrala.IDProvider {
	return &ulidProvider{
		entropy: rand.Reader,
	}
}

func (up *ulidProvider) ID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), up.entropy)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
