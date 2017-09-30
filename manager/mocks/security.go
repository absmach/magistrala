package mocks

import "github.com/mainflux/mainflux/manager"

var (
	_ manager.Hasher           = (*hasherMock)(nil)
	_ manager.IdentityProvider = (*identityProviderMock)(nil)
)

type hasherMock struct{}

func (hm *hasherMock) Hash(pwd string) (string, error) {
	return pwd, nil
}

func (hm *hasherMock) Compare(plain, hashed string) error {
	if plain != hashed {
		return manager.ErrUnauthorizedAccess
	}

	return nil
}

type identityProviderMock struct{}

func (idp *identityProviderMock) TemporaryKey(id string) (string, error) {
	if id == "" {
		return "", manager.ErrUnauthorizedAccess
	}

	return id, nil
}

func (idp *identityProviderMock) PermanentKey(id string) (string, error) {
	return idp.TemporaryKey(id)
}

func (idp *identityProviderMock) Identity(key string) (string, error) {
	return idp.TemporaryKey(key)
}

// NewHasher creates "no-op" hasher for test purposes. This implementation will
// return secrets without changing them.
func NewHasher() manager.Hasher {
	return &hasherMock{}
}

// NewIdentityProvider creates "mirror" identity provider, i.e. generated
// token will hold value provided by the caller.
func NewIdentityProvider() manager.IdentityProvider {
	return &identityProviderMock{}
}
