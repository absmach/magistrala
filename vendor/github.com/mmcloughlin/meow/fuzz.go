// +build gofuzz

package meow

import (
	"bytes"
	"encoding/binary"
)

func Fuzz(data []byte) int {
	if len(data) < 8 {
		return 0
	}
	seed := binary.BigEndian.Uint64(data)
	data = data[8:]

	expect := Checksum(seed, data)

	alt := []checksumFunc{
		checksumHash,
		checksumPureGo,
		checksumHashWithIntermediateSum,
	}

	for _, a := range alt {
		if !bytes.Equal(expect[:], a(seed, data)) {
			panic("mismatch")
		}
	}

	return 0
}
