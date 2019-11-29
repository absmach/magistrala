// Copyright (c) 2019 Faye Amacker. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cbor

import (
	"encoding/binary"
	"errors"
	"io"
	"strconv"
)

// SyntaxError is a description of a CBOR syntax error.
type SyntaxError struct {
	msg string
}

func (e *SyntaxError) Error() string { return e.msg }

// SemanticError is a description of a CBOR semantic error.
type SemanticError struct {
	msg string
}

func (e *SemanticError) Error() string { return e.msg }

// Valid checks whether CBOR data is complete and well-formed.
func Valid(data []byte) (rest []byte, err error) {
	if len(data) == 0 {
		return nil, io.EOF
	}
	offset, _, _, err := checkValid(data, 0)
	if err != nil {
		return nil, err
	}
	return data[offset:], nil
}

func checkValid(data []byte, off int) (_ int, t cborType, indefinite bool, err error) {
	if len(data)-off < 1 {
		return 0, 0, false, io.ErrUnexpectedEOF
	}

	var val uint64
	off, t, val, indefinite, err = checkTypeAndValue(data, off)
	if err != nil {
		return 0, 0, false, err
	}

	if indefinite {
		off, err = checkValidIndefinite(data, off, t)
		return off, t, indefinite, err
	}

	switch t {
	case cborTypeByteString, cborTypeTextString:
		valInt := int(val)
		if valInt < 0 {
			// Detect integer overflow
			return 0, 0, false, errors.New("cbor: " + t.String() + " length " + strconv.FormatUint(val, 10) + " is too large, causing integer overflow")
		}
		if len(data)-off < valInt {
			return 0, 0, false, io.ErrUnexpectedEOF
		}
		off += valInt
	case cborTypeArray, cborTypeMap:
		valInt := int(val)
		if valInt < 0 {
			// Detect integer overflow
			return 0, 0, false, errors.New("cbor: " + t.String() + " length " + strconv.FormatUint(val, 10) + " is too large, causing integer overflow")
		}
		count := 1
		if t == cborTypeMap {
			count = 2
		}
		for j := 0; j < count; j++ {
			for i := 0; i < valInt; i++ {
				if off, _, _, err = checkValid(data, off); err != nil {
					return 0, 0, false, err
				}
			}
		}
	case cborTypeTag: // Check tagged item following tag.
		return checkValid(data, off)
	}
	return off, t, indefinite, nil
}

func checkValidIndefinite(data []byte, off int, t cborType) (_ int, err error) {
	var nextType cborType
	var indefinite bool
	isByteOrTextString := (t == cborTypeByteString) || (t == cborTypeTextString)
	i := 0
	for ; true; i++ {
		if len(data)-off < 1 {
			return 0, io.ErrUnexpectedEOF
		}
		if data[off] == 0xFF {
			off++
			break
		}
		if off, nextType, indefinite, err = checkValid(data, off); err != nil {
			return 0, err
		}
		if isByteOrTextString {
			if t != nextType {
				return 0, &SemanticError{"cbor: wrong element type " + nextType.String() + " for indefinite-length " + t.String()}
			}
			if indefinite {
				return 0, &SemanticError{"cbor: indefinite-length " + t.String() + " chunk is not definite-length"}
			}
		}
	}
	if t == cborTypeMap && i%2 == 1 {
		return 0, &SyntaxError{"cbor: unexpected \"break\" code"}
	}
	return off, nil
}

func checkTypeAndValue(data []byte, off int) (_ int, t cborType, val uint64, indefinite bool, err error) {
	t = cborType(data[off] & 0xE0)
	ai := data[off] & 0x1F
	val = uint64(ai)
	off++

	switch ai {
	case 24:
		if len(data)-off < 1 {
			return 0, 0, 0, false, io.ErrUnexpectedEOF
		}
		val = uint64(data[off])
		off++
	case 25:
		if len(data)-off < 2 {
			return 0, 0, 0, false, io.ErrUnexpectedEOF
		}
		val = uint64(binary.BigEndian.Uint16(data[off : off+2]))
		off += 2
	case 26:
		if len(data)-off < 4 {
			return 0, 0, 0, false, io.ErrUnexpectedEOF
		}
		val = uint64(binary.BigEndian.Uint32(data[off : off+4]))
		off += 4
	case 27:
		if len(data)-off < 8 {
			return 0, 0, 0, false, io.ErrUnexpectedEOF
		}
		val = binary.BigEndian.Uint64(data[off : off+8])
		off += 8
	case 28, 29, 30:
		return 0, 0, 0, false, &SyntaxError{"cbor: invalid additional information " + strconv.Itoa(int(ai)) + " for type " + t.String()}
	case 31:
		switch t {
		case cborTypePositiveInt, cborTypeNegativeInt, cborTypeTag:
			return 0, 0, 0, false, &SyntaxError{"cbor: invalid additional information " + strconv.Itoa(int(ai)) + " for type " + t.String()}
		case cborTypePrimitives: // 0xFF (break code) should not be outside checkValidIndefinite().
			return 0, 0, 0, false, &SyntaxError{"cbor: unexpected \"break\" code"}
		default:
			return off, t, val, true, nil
		}
	}
	if t == cborTypePrimitives && ai == 24 && val < 32 {
		return 0, 0, 0, false, &SyntaxError{"cbor: invalid simple value " + strconv.Itoa(int(val)) + " for type " + t.String()}
	}
	return off, t, val, false, nil
}
