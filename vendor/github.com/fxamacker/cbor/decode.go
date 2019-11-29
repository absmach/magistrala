// Copyright (c) 2019 Faye Amacker. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cbor

import (
	"encoding"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Unmarshal parses the CBOR-encoded data and stores the result in the value
// pointed to by v.  If v is nil or not a pointer, Unmarshal returns an error.
//
// Unmarshal uses the inverse of the encodings that Marshal uses, allocating
// maps, slices, and pointers as necessary, with the following additional rules:
//
// To unmarshal CBOR into a pointer, Unmarshal first handles the case of the
// CBOR being the CBOR literal null.  In that case, Unmarshal sets the pointer
// to nil.  Otherwise, Unmarshal unmarshals the CBOR into the value pointed at
// by the pointer.  If the pointer is nil, Unmarshal allocates a new value for
// it to point to.
//
// To unmarshal CBOR into an interface value, Unmarshal stores one of these in
// the interface value:
//
//     bool, for CBOR booleans
//     uint64, for CBOR positive integers
//     int64, for CBOR negative integers
//     float64, for CBOR floating points
//     []byte, for CBOR byte strings
//     string, for CBOR text strings
//     []interface{}, for CBOR arrays
//     map[interface{}]interface{}, for CBOR maps
//     nil, for CBOR null
//
// To unmarshal a CBOR array into a slice, Unmarshal allocates a new slice only
// if the CBOR array is empty or slice capacity is less than CBOR array length.
// Otherwise Unmarshal reuses the existing slice, overwriting existing elements.
// Unmarshal sets the slice length to CBOR array length.
//
// To ummarshal a CBOR array into a Go array, Unmarshal decodes CBOR array
// elements into corresponding Go array elements.  If the Go array is smaller
// than the CBOR array, the additional CBOR array elements are discarded.  If
// the CBOR array is smaller than the Go array, the additional Go array elements
// are set to zero values.
//
// To unmarshal a CBOR map into a map, Unmarshal allocates a new map only if the
// map is nil.  Otherwise Unmarshal reuses the existing map, keeping existing
// entries.  Unmarshal stores key-value pairs from the CBOR map into Go map.
//
// To unmarshal a CBOR map into a struct, Unmarshal matches CBOR map keys to the
// keys in the following priority:
//
//     1. "cbor" key in struct field tag,
//     2. "json" key in struct field tag,
//     3. struct field name.
//
// Unmarshal prefers an exact match but also accepts a case-insensitive match.
// Map keys which don't have a corresponding struct field are ignored.
//
// To unmarshal a CBOR text string into a time.Time value, Unmarshal parses text
// string formatted in RFC3339.  To unmarshal a CBOR integer/float into a
// time.Time value, Unmarshal creates an unix time with integer/float as seconds
// and fractional seconds since January 1, 1970 UTC.
//
// To unmarshal CBOR into a value implementing the Unmarshaler interface,
// Unmarshal calls that value's UnmarshalCBOR method.
//
// Unmarshal decodes a CBOR byte string into a value implementing
// encoding.BinaryUnmarshaler.
//
// If a CBOR value is not appropriate for a given Go type, or if a CBOR number
// overflows the Go type, Unmarshal skips that field and completes the
// unmarshalling as best as it can.  If no more serious errors are encountered,
// unmarshal returns an UnmarshalTypeError describing the earliest such error.
// In any case, it's not guaranteed that all the remaining fields following the
// problematic one will be unmarshaled into the target object.
//
// The CBOR null value unmarshals into a slice/map/pointer/interface by setting
// that Go value to nil.  Because null is often used to mean "not present",
// unmarshalling a CBOR null into any other Go type has no effect on the value
// produces no error.
//
// Unmarshal ignores CBOR tag data and parses tagged data following CBOR tag.
func Unmarshal(data []byte, v interface{}) error {
	d := decodeState{data: data}
	return d.value(v)
}

// Unmarshaler is the interface implemented by types that can unmarshal a CBOR
// representation of themselves.  The input can be assumed to be a valid encoding
// of a CBOR value. UnmarshalCBOR must copy the CBOR data if it wishes to retain
// the data after returning.
type Unmarshaler interface {
	UnmarshalCBOR([]byte) error
}

// InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "cbor: Unmarshal(nil)"
	}
	if e.Type.Kind() != reflect.Ptr {
		return "cbor: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "cbor: Unmarshal(nil " + e.Type.String() + ")"
}

// UnmarshalTypeError describes a CBOR value that was not appropriate for a Go type.
type UnmarshalTypeError struct {
	Value  string       // description of CBOR value
	Type   reflect.Type // type of Go value it could not be assigned to
	Struct string       // struct type containing the field
	Field  string       // name of the field holding the Go value
	errMsg string       // additional error message (optional)
}

func (e *UnmarshalTypeError) Error() string {
	var s string
	if e.Struct != "" || e.Field != "" {
		s = "cbor: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String()
	} else {
		s = "cbor: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
	}
	if e.errMsg != "" {
		s += " (" + e.errMsg + ")"
	}
	return s
}

type decodeState struct {
	data []byte
	off  int // next read offset in data
}

func (d *decodeState) value(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{reflect.TypeOf(v)}
	}

	if _, err := Valid(d.data[d.off:]); err != nil {
		return err
	}

	rv = rv.Elem()

	if rv.Kind() == reflect.Interface && rv.NumMethod() == 0 && rv.IsNil() {
		// Fast path to decode to empty interface without calling implementsUnmarshaler.
		iv, err := d.parseInterface()
		if iv != nil {
			rv.Set(reflect.ValueOf(iv))
		}
		return err
	}

	return d.parse(rv, implementsUnmarshaler(rv.Type()))
}

type cborType uint8

const (
	cborTypePositiveInt cborType = 0x00
	cborTypeNegativeInt cborType = 0x20
	cborTypeByteString  cborType = 0x40
	cborTypeTextString  cborType = 0x60
	cborTypeArray       cborType = 0x80
	cborTypeMap         cborType = 0xA0
	cborTypeTag         cborType = 0xC0
	cborTypePrimitives  cborType = 0xE0
)

func (t cborType) String() string {
	switch t {
	case cborTypePositiveInt:
		return "positive integer"
	case cborTypeNegativeInt:
		return "negative integer"
	case cborTypeByteString:
		return "byte string"
	case cborTypeTextString:
		return "UTF-8 text string"
	case cborTypeArray:
		return "array"
	case cborTypeMap:
		return "map"
	case cborTypeTag:
		return "tag"
	case cborTypePrimitives:
		return "primitives"
	default:
		return "Invalid type " + strconv.Itoa(int(t))
	}
}

// parse assumes data is well-formed, and does not perform bounds checking.
func (d *decodeState) parse(v reflect.Value, isUnmarshaler bool) (err error) {
	if v.Kind() == reflect.Interface && v.NumMethod() == 0 && v.IsNil() {
		// nil interface
		iv, err := d.parseInterface()
		if err != nil {
			return err
		}
		if iv != nil {
			v.Set(reflect.ValueOf(iv))
		}
		return nil
	}

	// Create new value for the pointer v to point to if CBOR value is not nil/undefined.
	if d.data[d.off] != 0xf6 && d.data[d.off] != 0xf7 {
		for v.Kind() == reflect.Ptr {
			if v.IsNil() {
				if !v.CanSet() {
					return errors.New("cbor: cannot set new value for " + v.Type().String())
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
	}

	if isUnmarshaler {
		v1 := v
		if v1.Kind() != reflect.Ptr && v1.CanAddr() {
			v1 = v1.Addr()
		}
		if v1.Kind() == reflect.Ptr && !v1.IsNil() {
			if u, ok := v1.Interface().(Unmarshaler); ok {
				start := d.off
				d.skip()
				return u.UnmarshalCBOR(d.data[start:d.off])
			}
		}
	}

	// Process byte/text string.
	t := cborType(d.data[d.off] & 0xE0)
	if t == cborTypeByteString {
		b := d.parseByteString()
		return fillByteString(t, b, v)
	} else if t == cborTypeTextString {
		b, err := d.parseTextString()
		if err != nil {
			return err
		}
		return fillTextString(t, b, v)
	}

	t, ai, val := d.getHeader()

	// Process other types.
	switch t {
	case cborTypePositiveInt:
		return fillPositiveInt(t, val, v)
	case cborTypeNegativeInt:
		if val > math.MaxInt64 {
			return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: "-1-" + strconv.FormatUint(val, 10) + " overflows Go's int64"}
		}
		nValue := int64(-1) ^ int64(val)
		return fillNegativeInt(t, nValue, v)
	case cborTypeTag:
		return d.parse(v, isUnmarshaler)
	case cborTypePrimitives:
		if ai < 20 {
			return fillPositiveInt(t, uint64(ai), v)
		}
		switch ai {
		case 20, 21:
			return fillBool(t, ai == 21, v)
		case 22, 23:
			return fillNil(t, v)
		case 24:
			return fillPositiveInt(t, uint64(val), v)
		case 25:
			f := uint16ToFloat64(uint16(val))
			return fillFloat(t, f, v)
		case 26:
			f := float64(math.Float32frombits(uint32(val)))
			return fillFloat(t, f, v)
		case 27:
			f := math.Float64frombits(val)
			return fillFloat(t, f, v)
		}
	case cborTypeArray:
		valInt := int(val)
		count := valInt
		if ai == 31 {
			count = -1
		}
		if v.Kind() == reflect.Slice {
			return d.parseSlice(t, count, v)
		} else if v.Kind() == reflect.Array {
			return d.parseArray(t, count, v)
		} else if v.Kind() == reflect.Struct {
			return d.parseStructFromArray(t, count, v)
		}
		hasSize := count >= 0
		for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
			d.skip()
		}
		return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
	case cborTypeMap:
		valInt := int(val)
		count := valInt
		if ai == 31 {
			count = -1
		}
		if v.Kind() == reflect.Struct {
			return d.parseStructFromMap(t, count, v)
		} else if v.Kind() == reflect.Map {
			return d.parseMap(t, count, v)
		}
		hasSize := count >= 0
		for i := 0; (hasSize && i < count*2) || (!hasSize && !d.foundBreak()); i++ {
			d.skip()
		}
		return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
	}
	return nil
}

// parseInterface assumes data is well-formed, and does not perform bounds checking.
func (d *decodeState) parseInterface() (_ interface{}, err error) {
	// Process byte/text string.
	t := cborType(d.data[d.off] & 0xE0)
	if t == cborTypeByteString {
		return d.parseByteString(), nil
	} else if t == cborTypeTextString {
		b, err := d.parseTextString()
		if err != nil {
			return nil, err
		}
		return string(b), nil
	}

	t, ai, val := d.getHeader()

	// Process other types.
	switch t {
	case cborTypePositiveInt:
		return val, nil
	case cborTypeNegativeInt:
		if val > math.MaxInt64 {
			return nil, &UnmarshalTypeError{Value: t.String(), Type: reflect.TypeOf([]interface{}(nil)).Elem(), errMsg: "-1-" + strconv.FormatUint(val, 10) + " overflows Go's int64"}
		}
		nValue := int64(-1) ^ int64(val)
		return nValue, nil
	case cborTypeTag:
		return d.parseInterface()
	case cborTypePrimitives:
		if ai < 20 {
			return uint64(ai), nil
		}
		switch ai {
		case 20, 21:
			return (ai == 21), nil
		case 22, 23:
			return nil, nil
		case 24:
			return uint64(val), nil
		case 25:
			f := uint16ToFloat64(uint16(val))
			return f, nil
		case 26:
			f := float64(math.Float32frombits(uint32(val)))
			return f, nil
		case 27:
			f := math.Float64frombits(val)
			return f, nil
		}
	case cborTypeArray:
		count := int(val)
		if ai == 31 {
			count = -1
		}
		return d.parseArrayInterface(t, count)
	case cborTypeMap:
		count := int(val)
		if ai == 31 {
			count = -1
		}
		return d.parseMapInterface(t, count)
	}
	return nil, nil
}

// parseByteString parses CBOR encoded byte string.  It returns a byte slice
// pointing to a copy of parsed data.
func (d *decodeState) parseByteString() []byte {
	val, isCopy := d.parseStringBuf(nil)
	if !isCopy {
		// Make a copy of val so that GC can collect underlying data val points to.
		copyVal := make([]byte, len(val))
		copy(copyVal, val)
		return copyVal
	}
	return val
}

// parseTextString parses CBOR encoded text string.  It does not return a string
// to prevent creating an extra copy of string.  Caller should wrap returned
// byte slice as string when needed.
//
// parseStruct() uses parseTextString() to improve memory and performance,
// compared with using parse(reflect.Value).  parse(reflect.Value) sets
// reflect.Value with parsed string, while parseTextString() returns parsed string.
func (d *decodeState) parseTextString() ([]byte, error) {
	val, _ := d.parseStringBuf(nil)

	if !utf8.Valid(val) {
		return nil, &SemanticError{"cbor: invalid UTF-8 string"}
	}

	return val, nil
}

// parseStringBuf assumes data is well-formed, and does not perform bounds checking.
func (d *decodeState) parseStringBuf(p []byte) (_ []byte, isCopy bool) {
	t, ai, val := d.getHeader()

	if t == cborTypeTag {
		return d.parseStringBuf(p)
	}

	if ai == 31 {
		// Process indefinite length string.
		if p == nil {
			p = make([]byte, 0, 64)
		}
		for !d.foundBreak() {
			p, _ = d.parseStringBuf(p)
		}
		return p, true
	}

	// Process definite length string.
	oldOff, newOff := d.off, d.off+int(val)
	d.off = newOff

	if p != nil {
		p = append(p, d.data[oldOff:newOff]...)
		return p, true
	}
	return d.data[oldOff:newOff], false
}

func (d *decodeState) parseArrayInterface(t cborType, count int) ([]interface{}, error) {
	hasSize := count >= 0
	if count == -1 {
		count = d.numOfItemsUntilBreak() // peek ahead to get array size to preallocate slice for better performance
	}
	v := make([]interface{}, count)
	var e interface{}
	var err, lastErr error
	for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
		if e, lastErr = d.parseInterface(); lastErr != nil {
			if err == nil {
				err = lastErr
			}
			continue
		}
		v[i] = e
	}
	return v, err
}

func (d *decodeState) parseSlice(t cborType, count int, v reflect.Value) error {
	hasSize := count >= 0
	if count == -1 {
		count = d.numOfItemsUntilBreak() // peek ahead to get array size to preallocate slice for better performance
	}
	if count == 0 {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}
	if v.IsNil() || v.Cap() < count {
		v.Set(reflect.MakeSlice(v.Type(), count, count))
	}
	v.SetLen(count)
	elemIsUnmarshaler := implementsUnmarshaler(v.Type().Elem())
	var err error
	for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
		if lastErr := d.parse(v.Index(i), elemIsUnmarshaler); lastErr != nil {
			if err == nil {
				err = lastErr
			}
		}
	}
	return err
}

func (d *decodeState) parseArray(t cborType, count int, v reflect.Value) error {
	hasSize := count >= 0
	elemIsUnmarshaler := implementsUnmarshaler(v.Type().Elem())
	i := 0
	var err error
	for ; i < v.Len() && ((hasSize && i < count) || (!hasSize && !d.foundBreak())); i++ {
		if lastErr := d.parse(v.Index(i), elemIsUnmarshaler); lastErr != nil {
			if err == nil {
				err = lastErr
			}
		}
	}
	// Set remaining Go array elements to zero values.
	if i < v.Len() {
		zeroV := reflect.Zero(v.Type().Elem())
		for ; i < v.Len(); i++ {
			v.Index(i).Set(zeroV)
		}
	}
	// Skip remaining CBOR array elements
	for ; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
		d.skip()
	}
	return err
}

func (d *decodeState) parseMapInterface(t cborType, count int) (map[interface{}]interface{}, error) {
	m := make(map[interface{}]interface{})
	hasSize := count >= 0
	var k, e interface{}
	var err, lastErr error
	for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
		if k, lastErr = d.parseInterface(); lastErr != nil {
			if err == nil {
				err = lastErr
			}
			d.skip()
			continue
		}
		kkind := reflect.ValueOf(k).Kind()
		if !isHashableKind(kkind) {
			if err == nil {
				err = errors.New("cbor: invalid map key type: " + kkind.String())
			}
			d.skip()
			continue
		}
		if e, lastErr = d.parseInterface(); lastErr != nil {
			if err == nil {
				err = lastErr
			}
			continue
		}
		m[k] = e
	}
	return m, err
}

func (d *decodeState) parseMap(t cborType, count int, v reflect.Value) error {
	if v.IsNil() {
		mapsize := count
		if mapsize < 0 {
			mapsize = 0
		}
		v.Set(reflect.MakeMapWithSize(v.Type(), mapsize))
	}
	hasSize := count >= 0
	keyType, eleType := v.Type().Key(), v.Type().Elem()
	reuseKey, reuseEle := isImmutableKind(keyType.Kind()), isImmutableKind(eleType.Kind())
	var keyValue, eleValue, zeroKeyValue, zeroEleValue reflect.Value
	keyIsUnmarshaler := implementsUnmarshaler(v.Type().Key())
	elemIsUnmarshaler := implementsUnmarshaler(v.Type().Elem())
	keyIsInterfaceType := keyType == typeIntf // If key type is interface{}, need to check if key value is hashable.
	var err, lastErr error
	for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
		if !keyValue.IsValid() {
			keyValue = reflect.New(keyType).Elem()
		} else if !reuseKey {
			if !zeroKeyValue.IsValid() {
				zeroKeyValue = reflect.Zero(keyType)
			}
			keyValue.Set(zeroKeyValue)
		}
		if lastErr = d.parse(keyValue, keyIsUnmarshaler); lastErr != nil {
			if err == nil {
				err = lastErr
			}
			d.skip()
			continue
		}
		if keyIsInterfaceType && !isHashableKind(keyValue.Elem().Kind()) {
			if err == nil {
				err = errors.New("cbor: invalid map key type: " + keyValue.Elem().Kind().String())
			}
			d.skip()
			continue
		}

		if !eleValue.IsValid() {
			eleValue = reflect.New(eleType).Elem()
		} else if !reuseEle {
			if !zeroEleValue.IsValid() {
				zeroEleValue = reflect.Zero(eleType)
			}
			eleValue.Set(zeroEleValue)
		}
		if lastErr := d.parse(eleValue, elemIsUnmarshaler); lastErr != nil {
			if err == nil {
				err = lastErr
			}
			continue
		}

		v.SetMapIndex(keyValue, eleValue)
	}
	return err
}

func (d *decodeState) parseStructFromArray(t cborType, count int, v reflect.Value) error {
	structType := getDecodingStructType(v.Type())
	if !structType.toArray {
		hasSize := count >= 0
		for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
			d.skip()
		}
		return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: "cannot decode CBOR array to struct without toarray option"}
	}
	hasSize := count >= 0
	if count == -1 {
		count = d.numOfItemsUntilBreak() // peek ahead to get array size to verify that array size matches number of fields
	}
	if count != len(structType.fields) {
		for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
			d.skip()
		}
		return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: "cannot decode CBOR array to struct with different number of elements"}
	}
	var err error
	for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
		fv, lastErr := fieldByIndex(v, structType.fields[i].idx)
		if lastErr != nil {
			if err == nil {
				err = lastErr
			}
			d.skip()
			continue
		}
		if lastErr := d.parse(fv, structType.fields[i].isUnmarshaler); lastErr != nil {
			if err == nil {
				if typeError, ok := lastErr.(*UnmarshalTypeError); ok {
					typeError.Struct = v.Type().String()
					typeError.Field = structType.fields[i].name
					err = typeError
				} else {
					err = lastErr
				}
			}
		}
	}
	return err
}

func (d *decodeState) parseStructFromMap(t cborType, count int, v reflect.Value) error {
	structType := getDecodingStructType(v.Type())

	foundFldIdx := make([]bool, len(structType.fields))
	hasSize := count >= 0
	var err, lastErr error
	for i := 0; (hasSize && i < count) || (!hasSize && !d.foundBreak()); i++ {
		var keyBytes []byte
		t := cborType(d.data[d.off] & 0xE0)
		if t == cborTypeTextString {
			keyBytes, lastErr = d.parseTextString()
			if lastErr != nil {
				if err == nil {
					err = lastErr
				}
				d.skip() // skip value
				continue
			}
		} else if t == cborTypePositiveInt || t == cborTypeNegativeInt {
			iv, lastErr := d.parseInterface()
			if lastErr != nil {
				if err == nil {
					err = lastErr
				}
				d.skip() // skip value
				continue
			}
			switch n := iv.(type) {
			case int64:
				keyBytes = []byte(strconv.Itoa(int(n)))
			case uint64:
				keyBytes = []byte(strconv.Itoa(int(n)))
			}
		} else {
			if err == nil {
				err = &UnmarshalTypeError{Value: t.String(), Type: reflect.TypeOf(""), errMsg: "map key is of type " + t.String() + " and cannot be used to match struct " + v.Type().String() + " field name"}
			}
			d.skip() // skip key
			d.skip() // skip value
			continue
		}
		keyLen := len(keyBytes)

		var f *field
		for i := 0; i < len(structType.fields); i++ {
			// Find field with exact match
			if !foundFldIdx[i] && len(structType.fields[i].name) == keyLen && structType.fields[i].name == string(keyBytes) {
				f = &structType.fields[i]
				foundFldIdx[i] = true
				break
			}
		}
		if f == nil {
			keyString := string(keyBytes)
			for i := 0; i < len(structType.fields); i++ {
				// Find field with case-insensitive match
				if !foundFldIdx[i] && len(structType.fields[i].name) == keyLen && strings.EqualFold(structType.fields[i].name, keyString) {
					f = &structType.fields[i]
					foundFldIdx[i] = true
					break
				}
			}
		}
		if f == nil {
			d.skip()
			continue
		}
		// reflect.Value.FieldByIndex() panics at nil pointer to unexported
		// anonymous field.  fieldByIndex() returns error.
		fv, lastErr := fieldByIndex(v, f.idx)
		if lastErr != nil {
			if err == nil {
				err = lastErr
			}
			d.skip()
			continue
		}
		if lastErr = d.parse(fv, f.isUnmarshaler); lastErr != nil {
			if err == nil {
				if typeError, ok := lastErr.(*UnmarshalTypeError); ok {
					typeError.Struct = v.Type().String()
					typeError.Field = f.name
					err = typeError
				} else {
					err = lastErr
				}
			}
		}
	}
	return err
}

// skip moves data offset to the next item.  skip assumes data is well-formed,
// and does not perform bounds checking.
func (d *decodeState) skip() {
	t := cborType(d.data[d.off] & 0xE0)
	ai := d.data[d.off] & 0x1F
	val := uint64(ai)
	d.off++

	switch ai {
	case 24:
		val = uint64(d.data[d.off])
		d.off++
	case 25:
		val = uint64(binary.BigEndian.Uint16(d.data[d.off : d.off+2]))
		d.off += 2
	case 26:
		val = uint64(binary.BigEndian.Uint32(d.data[d.off : d.off+4]))
		d.off += 4
	case 27:
		val = binary.BigEndian.Uint64(d.data[d.off : d.off+8])
		d.off += 8
	}

	if ai == 31 {
		switch t {
		case cborTypeByteString, cborTypeTextString, cborTypeArray, cborTypeMap:
			for true {
				if d.data[d.off] == 0xFF {
					d.off++
					return
				}
				d.skip()
			}
		}
	}

	switch t {
	case cborTypeByteString, cborTypeTextString:
		d.off += int(val)
	case cborTypeArray:
		for i := 0; i < int(val); i++ {
			d.skip()
		}
	case cborTypeMap:
		for i := 0; i < int(val)*2; i++ {
			d.skip()
		}
	case cborTypeTag:
		d.skip()
	}
}

// getHeader assumes data is well-formed, and does not perform bounds checking.
func (d *decodeState) getHeader() (t cborType, ai byte, val uint64) {
	t = cborType(d.data[d.off] & 0xE0)
	ai = d.data[d.off] & 0x1F
	val = uint64(ai)
	d.off++

	switch ai {
	case 24:
		val = uint64(d.data[d.off])
		d.off++
	case 25:
		val = uint64(binary.BigEndian.Uint16(d.data[d.off : d.off+2]))
		d.off += 2
	case 26:
		val = uint64(binary.BigEndian.Uint32(d.data[d.off : d.off+4]))
		d.off += 4
	case 27:
		val = binary.BigEndian.Uint64(d.data[d.off : d.off+8])
		d.off += 8
	}
	return
}

func (d *decodeState) numOfItemsUntilBreak() int {
	savedOff := d.off
	i := 0
	for !d.foundBreak() {
		d.skip()
		i++
	}
	d.off = savedOff
	return i
}

// foundBreak assumes data is well-formed, and does not perform bounds checking.
func (d *decodeState) foundBreak() bool {
	if d.data[d.off] == 0xFF {
		d.off++
		return true
	}
	return false
}

func (d *decodeState) reset(data []byte) {
	d.data = data
	d.off = 0
}

var (
	typeIntf              = reflect.TypeOf([]interface{}(nil)).Elem()
	typeTime              = reflect.TypeOf(time.Time{})
	typeUnmarshaler       = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	typeBinaryUnmarshaler = reflect.TypeOf((*encoding.BinaryUnmarshaler)(nil)).Elem()
)

func fillNil(t cborType, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
		v.Set(reflect.Zero(v.Type()))
		return nil
	}
	if v.Type() == typeTime {
		v.Set(reflect.ValueOf(time.Time{}))
		return nil
	}
	return nil
}

func fillPositiveInt(t cborType, val uint64, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val > math.MaxInt64 {
			return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: strconv.FormatUint(val, 10) + " overflows " + v.Type().String()}
		}
		if v.OverflowInt(int64(val)) {
			return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: strconv.FormatUint(val, 10) + " overflows " + v.Type().String()}
		}
		v.SetInt(int64(val))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.OverflowUint(val) {
			return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: strconv.FormatUint(val, 10) + " overflows " + v.Type().String()}
		}
		v.SetUint(val)
		return nil
	case reflect.Float32, reflect.Float64:
		f := float64(val)
		v.SetFloat(f)
		return nil
	}
	if v.Type() == typeTime {
		tm := time.Unix(int64(val), 0)
		v.Set(reflect.ValueOf(tm))
		return nil
	}
	return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
}

func fillNegativeInt(t cborType, val int64, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.OverflowInt(val) {
			return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: strconv.FormatInt(val, 10) + " overflows " + v.Type().String()}
		}
		v.SetInt(val)
		return nil
	case reflect.Float32, reflect.Float64:
		f := float64(val)
		v.SetFloat(f)
		return nil
	}
	if v.Type() == typeTime {
		tm := time.Unix(val, 0)
		v.Set(reflect.ValueOf(tm))
		return nil
	}
	return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
}

func fillBool(t cborType, val bool, v reflect.Value) error {
	if v.Kind() == reflect.Bool {
		v.SetBool(val)
		return nil
	}
	return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
}

func fillFloat(t cborType, val float64, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		if v.OverflowFloat(val) {
			return &UnmarshalTypeError{Value: t.String(), Type: v.Type(), errMsg: strconv.FormatFloat(val, 'E', -1, 64) + " overflows " + v.Type().String()}
		}
		v.SetFloat(val)
		return nil
	}
	if v.Type() == typeTime {
		f1, f2 := math.Modf(val)
		tm := time.Unix(int64(f1), int64(f2*1e9))
		v.Set(reflect.ValueOf(tm))
		return nil
	}
	return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
}

func fillByteString(t cborType, val []byte, v reflect.Value) error {
	if v.Type() != typeTime && reflect.PtrTo(v.Type()).Implements(typeBinaryUnmarshaler) {
		if v.CanAddr() {
			v = v.Addr()
			if u, ok := v.Interface().(encoding.BinaryUnmarshaler); ok {
				return u.UnmarshalBinary(val)
			}
		}
		return errors.New("cbor: cannot set new value for " + v.Type().String())
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		v.SetBytes(val)
		return nil
	}
	return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
}

func fillTextString(t cborType, val []byte, v reflect.Value) error {
	if v.Kind() == reflect.String {
		v.SetString(string(val))
		return nil
	}
	if v.Type() == typeTime {
		tm, err := time.Parse(time.RFC3339, string(val))
		if err != nil {
			return errors.New("cbor: cannot set " + string(val) + " for time.Time")
		}
		v.Set(reflect.ValueOf(tm))
		return nil
	}
	return &UnmarshalTypeError{Value: t.String(), Type: v.Type()}
}

func uint16ToFloat64(num uint16) float64 {
	bits := uint32(num)

	sign := bits >> 15
	exp := bits >> 10 & 0x1F
	frac := bits & 0x3FF

	switch exp {
	case 0:
	case 0x1F:
		exp = 0xFF
	default:
		exp = exp - 15 + 127
	}
	bits = sign<<31 | exp<<23 | frac<<13

	f := math.Float32frombits(bits)
	return float64(f)
}

func isImmutableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

func isHashableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Slice, reflect.Map, reflect.Func:
		return false
	default:
		return true
	}
}

func implementsUnmarshaler(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return reflect.PtrTo(t).Implements(typeUnmarshaler)
}

// fieldByIndex returns the nested field corresponding to the index.  It
// allocates pointer to struct field if it is nil and settable.
// reflect.Value.FieldByIndex() panics at nil pointer to unexported anonymous
// field.  This function returns error.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, error) {
	for _, i := range index {
		if v.Kind() == reflect.Ptr && v.Type().Elem().Kind() == reflect.Struct {
			if v.IsNil() {
				if !v.CanSet() {
					return reflect.Value{}, errors.New("cbor: cannot set embedded pointer to unexported struct: " + v.Type().String())
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, nil
}
