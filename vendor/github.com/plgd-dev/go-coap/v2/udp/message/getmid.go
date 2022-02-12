package message

import (
	"crypto/rand"
	"encoding/binary"
	mathRand "math/rand"
	"sync/atomic"
	"time"
)

func init() {
	mathRand.Seed(time.Now().UnixNano())
}

var msgID = uint32(RandMID())

// GetMID generates a message id for UDP-coap
func GetMID() uint16 {
	return uint16(atomic.AddUint32(&msgID, 1))
}

func RandMID() uint16 {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		// fallback to cryptographically insecure pseudo-random generator
		return uint16(mathRand.Uint32() >> 16)
	}
	return uint16(binary.BigEndian.Uint32(b))
}
