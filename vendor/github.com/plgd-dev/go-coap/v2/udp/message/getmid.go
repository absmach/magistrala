package message

import (
	"crypto/rand"
	"encoding/binary"
	"sync/atomic"
)

var msgID uint32

func init() {
	b := make([]byte, 4)
	rand.Read(b)
	msgID = binary.BigEndian.Uint32(b)
}

// GetMID generates a message id for UDP-coap
func GetMID() uint16 {
	return uint16(atomic.AddUint32(&msgID, 1))
}
