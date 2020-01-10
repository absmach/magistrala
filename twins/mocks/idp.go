// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/twins"
)

const u4Pref = "123e4567-e89b-12d3-a456-"

var _ twins.IdentityProvider = (*identityProviderMock)(nil)

type identityProviderMock struct {
	mu      sync.Mutex
	counter int
}

func (idp *identityProviderMock) ID() (string, error) {
	idp.mu.Lock()
	defer idp.mu.Unlock()

	idp.counter++
	return fmt.Sprintf("%s%012d", u4Pref, idp.counter), nil
}

func (idp *identityProviderMock) IsValid(u4 string) error {
	if !strings.Contains(u4Pref, u4) {
		return twins.ErrMalformedEntity
	}

	return nil
}

// NewIdentityProvider creates "mirror" identity provider, i.e. generated
// token will hold value provided by the caller.
func NewIdentityProvider() twins.IdentityProvider {
	return &identityProviderMock{}
}
