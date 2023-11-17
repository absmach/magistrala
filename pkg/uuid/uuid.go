// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package uuid provides a UUID identity provider.
package uuid

import (
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/gofrs/uuid"
)

// ErrGeneratingID indicates error in generating UUID.
var ErrGeneratingID = errors.New("failed to generate uuid")

var _ magistrala.IDProvider = (*uuidProvider)(nil)

type uuidProvider struct{}

// New instantiates a UUID provider.
func New() magistrala.IDProvider {
	return &uuidProvider{}
}

func (up *uuidProvider) ID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(ErrGeneratingID, err)
	}

	return id.String(), nil
}
