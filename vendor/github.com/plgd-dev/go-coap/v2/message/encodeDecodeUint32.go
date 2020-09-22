package message

import (
	"encoding/binary"
)

func EncodeUint32(buf []byte, value uint32) (int, error) {
	switch {
	case value == 0:
		return 0, nil
	case value <= max1ByteNumber:
		if len(buf) < 1 {
			return 1, ErrTooSmall
		}
		buf[0] = byte(value)
		return 1, nil
	case value <= max2ByteNumber:
		if len(buf) < 2 {
			return 2, ErrTooSmall
		}
		binary.BigEndian.PutUint16(buf, uint16(value))
		return 2, nil
	case value <= max3ByteNumber:
		if len(buf) < 3 {
			return 3, ErrTooSmall
		}
		rv := make([]byte, 4)
		binary.BigEndian.PutUint32(rv[:], value)
		copy(buf, rv[1:])
		return 3, nil
	default:
		if len(buf) < 4 {
			return 4, ErrTooSmall
		}
		binary.BigEndian.PutUint32(buf, value)
		return 4, nil
	}
}

func DecodeUint32(buf []byte) (uint32, int, error) {
	if len(buf) > 4 {
		buf = buf[:4]
	}
	tmp := []byte{0, 0, 0, 0}
	copy(tmp[4-len(buf):], buf)
	value := binary.BigEndian.Uint32(tmp)
	return value, len(buf), nil
}
