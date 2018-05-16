package mocks

import (
	"fmt"
	"sync"

	"github.com/mainflux/mainflux/things"
)

var _ things.IdentityProvider = (*identityProviderMock)(nil)

const startID = "123e4567-e89b-12d3-a456-"

type identityProviderMock struct {
	mu      sync.Mutex
	counter int
}

func (idp *identityProviderMock) ID() string {
	idp.mu.Lock()
	defer idp.mu.Unlock()

	idp.counter++
	return fmt.Sprintf("%s%012d", startID, idp.counter)
}

// NewIdentityProvider creates "mirror" identity provider, i.e. generated
// token will hold value provided by the caller.
func NewIdentityProvider() things.IdentityProvider {
	return &identityProviderMock{}
}
