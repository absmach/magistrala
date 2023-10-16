package message

import (
	"crypto/rand"
	"encoding/hex"
	"hash/crc64"
)

type Token []byte

func (t Token) String() string {
	return hex.EncodeToString(t)
}

func (t Token) Hash() uint64 {
	return crc64.Checksum(t, crc64.MakeTable(crc64.ISO))
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
