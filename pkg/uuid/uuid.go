// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package uuid provides a UUID identity provider.
package uuid

import (
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/gofrs/uuid/v5"
)

// ErrGeneratingID indicates error in generating UUID.
var ErrGeneratingID = errors.New("failed to generate uuid")

var _ supermq.IDProvider = (*uuidProvider)(nil)

type uuidProvider struct{}

// New instantiates a UUID provider.
func New() supermq.IDProvider {
	return &uuidProvider{}
}

func (up *uuidProvider) ID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(ErrGeneratingID, err)
	}

	return id.String(), nil
}
