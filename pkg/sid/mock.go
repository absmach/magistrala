// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sid

import (
	"fmt"
	"sync"

	"github.com/absmach/magistrala"
)

// Prefix represents the prefix used to generate UUID mocks.
const Prefix = "123e4567-e89b-12d3-a456-"

var _ magistrala.IDProvider = (*sidProviderMock)(nil)

type sidProviderMock struct {
	mu      sync.Mutex
	counter int
}

func (up *sidProviderMock) ID() (string, error) {
	up.mu.Lock()
	defer up.mu.Unlock()

	up.counter++
	return fmt.Sprintf("%s%012d", Prefix, up.counter), nil
}

// NewMock creates "mirror" uuid provider, i.e. generated
// token will hold value provided by the caller.
func NewMock() magistrala.IDProvider {
	return &sidProviderMock{}
}
