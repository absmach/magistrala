package bcrypt

import (
	"github.com/mainflux/mainflux/manager"
	"golang.org/x/crypto/bcrypt"
)

const cost int = 10

var _ manager.Hasher = (*bcryptHasher)(nil)

type bcryptHasher struct{}

// NewHasher instantiates a bcrypt-based hasher implementation.
func NewHasher() manager.Hasher {
	return &bcryptHasher{}
}

func (bh *bcryptHasher) Hash(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (bh *bcryptHasher) Compare(plain, hashed string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
