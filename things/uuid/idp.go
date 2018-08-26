//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package uuid provides a UUID identity provider.
package uuid

import (
	"github.com/mainflux/mainflux/things"
	uuid "github.com/satori/go.uuid"
)

var _ things.IdentityProvider = (*uuidIdentityProvider)(nil)

type uuidIdentityProvider struct{}

// New instantiates a UUID identity provider.
func New() things.IdentityProvider {
	return &uuidIdentityProvider{}
}

func (idp *uuidIdentityProvider) ID() string {
	return uuid.NewV4().String()
}
