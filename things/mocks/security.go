package mocks

import (
	"github.com/mainflux/mainflux/things"
)

var _ things.IdentityProvider = (*identityProviderMock)(nil)

type identityProviderMock struct{}

func (idp *identityProviderMock) TemporaryKey(id string) (string, error) {
	if id == "" {
		return "", things.ErrUnauthorizedAccess
	}

	return id, nil
}

func (idp *identityProviderMock) PermanentKey(id string) (string, error) {
	return idp.TemporaryKey(id)
}

func (idp *identityProviderMock) Identity(key string) (string, error) {
	return idp.TemporaryKey(key)
}

// NewIdentityProvider creates "mirror" identity provider, i.e. generated
// token will hold value provided by the caller.
func NewIdentityProvider() things.IdentityProvider {
	return &identityProviderMock{}
}
