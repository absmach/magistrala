// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package uuid provides a UUID identity provider.
package uuid

import (
	"github.com/gofrs/uuid"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
)

// ErrGeneratingID indicates error in generating UUID
var ErrGeneratingID = errors.New("failed to generate uuid")

var _ mainflux.IDProvider = (*uuidProvider)(nil)

type uuidProvider struct{}

// New instantiates a UUID provider.
func New() mainflux.IDProvider {
	return &uuidProvider{}
}

func (up *uuidProvider) ID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(ErrGeneratingID, err)
	}

	return id.String(), nil
}
