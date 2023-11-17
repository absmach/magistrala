// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package ulid provides a ULID identity provider.
package ulid

import (
	"math/rand"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/oklog/ulid/v2"
)

// ErrGeneratingID indicates error in generating ULID.
var ErrGeneratingID = errors.New("generating id failed")

var _ magistrala.IDProvider = (*ulidProvider)(nil)

type ulidProvider struct {
	entropy *rand.Rand
}

// New instantiates a ULID provider.
func New() magistrala.IDProvider {
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	return &ulidProvider{
		entropy: rand.New(source),
	}
}

func (up *ulidProvider) ID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), up.entropy)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
