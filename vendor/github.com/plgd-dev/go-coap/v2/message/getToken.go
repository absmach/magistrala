package message

import (
	"crypto/rand"
	"encoding/hex"
)

type Token []byte

func (t Token) String() string {
	return hex.EncodeToString(t)
}

// GetToken generates a random token by a given length
func GetToken() (Token, error) {
	b := make(Token, 8)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}
