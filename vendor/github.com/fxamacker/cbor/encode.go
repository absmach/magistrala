// Copyright (c) 2019 Faye Amacker. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cbor

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"math"
	"reflect"
	"sort"
	"sync"
	"time"
)

// Marshal returns the CBOR encoding of v.
//
// Marshal uses the following type-dependent default encodings:
//
// Boolean values encode as CBOR booleans (type 7).
//
// Positive integer values encode as CBOR positive integers (type 0).
//
// Negative integer values encode as CBOR negative integers (type 1).
//
// Floating point values encode as CBOR floating points (type 7).
//
// String values encode as CBOR text strings (type 3).
//
// []byte values encode as CBOR byte strings (type 2).
//
// Array and slice values encode as CBOR arrays (type 4).
//
// Map values encode as CBOR maps (type 5).
//
// Struct values encode as CBOR maps (type 5).  Each exported struct field
// becomes a pair with field name encoded as CBOR text string (type 3) and
// field value encoded based on its type.
//
// Pointer values encode as the value pointed to.
//
// Nil slice/map/pointer/interface values encode as CBOR nulls (type 7).
//
// time.Time values encode as text strings specified in RFC3339 when
// EncOptions.TimeRFC3339 is true; otherwise, time.Time values encode as
// numerical representation of seconds since January 1, 1970 UTC.
//
// If value implements the Marshaler interface, Marshal calls its MarshalCBOR
// method.  If value implements encoding.BinaryMarshaler instead, Marhsal
// calls its MarshalBinary method and encode it as CBOR byte string.
//
// Marshal supports format string stored under the "cbor" key in the struct
// field's tag.  CBOR format string can specify the name of the field, "omitempty"
// and "keyasint" options, and special case "-" for field omission.  If "cbor"
// key is absent, Marshal uses "json" key.
//
// Struct field name is treated as integer if it has "keyasint" option in
// its format string.  The format string must specify an integer as its
// field name.
//
// Special struct field "_" is used to specify struct level options, such as
// "toarray". "toarray" option enables Go struct to be encoded as CBOR array.
// "omitempty" is disabled by "toarray" to ensure that the same number
// of elements are encoded every time.
//
// Anonymous struct fields are usually marshalled as if their exported fields
// were fields in the outer struct.  Marshal follows the same struct fields
// visibility rules used by JSON encoding package.  An anonymous struct field
// with a name given in its CBOR tag is treated as having that name, rather
// than being anonymous.  An anonymous struct field of interface type is
// treated the same as having that type as its name, rather than being anonymous.
//
// Interface values encode as the value contained in the interface.  A nil
// interface value encodes as the null CBOR value.
//
// Channel, complex, and functon values cannot be encoded in CBOR.  Attempting
// to encode such a value causes Marshal to return an UnsupportedTypeError.
//
// Canonical CBOR encoding uses the following rules:
//
//     1. Integers must be as small as possible.
//     2. The expression of lengths in major types 2 through 5 must be as short
//        as possible.
//     3. The keys in every map must be sorted by the following rules:
//        *  If two keys have different lengths, the shorter one sorts earlier;
//		  *  If two keys have the same length, the one with the lower value in
//           (byte-wise) lexical order sorts earlier.
//     4. Indefinite-length items must be made into definite-length items.
//
// CTAP2 canonical CBOR encoding uses the following rules:
//
//     1. Integers must be encoded as small as possible.
//     2. The representations of any floating-point values are not changed.
//     3. The expression of lengths in major types 2 through 5 must be as short
//        as possible.
//     4. Indefinite-length items must be made into definite-length items.
//     5. The keys in every map must be sorted lowest value to highest.
//        * If the major types are different, the one with the lower value in
//          numerical order sorts earlier.
//        * If two keys have different lengths, the shorter one sorts earlier;
//        * If two keys have the same length, the one with the lower value in
//          (byte-wise) lexical order sorts earlier.
//     6. Tags must not be present.
//
// Marshal supports 2 options for canonical encoding:
// 1. Canonical: Canonical CBOR encoding (RFC 7049)
// 2. CTAP2Canonical:  CTAP2 canonical CBOR encoding
func Marshal(v interface{}, encOpts EncOptions) ([]byte, error) {
	e := getEncodeState()

	err := e.marshal(v, encOpts)
	if err != nil {
		putEncodeState(e)
		return nil, err
	}

	buf := make([]byte, e.Len())
	copy(buf, e.Bytes())

	putEncodeState(e)
	return buf, nil
}

// Marshaler is the interface implemented by types that can marshal themselves
// into valid CBOR.
type Marshaler interface {
	MarshalCBOR() ([]byte, error)
}

// UnsupportedTypeError is returned by Marshal when attempting to encode an
// unsupported value type.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "cbor: unsupported type: " + e.Type.String()
}

// EncOptions specifies encoding options.
type EncOptions struct {
	// Canonical causes map and struct to be encoded in a predictable sequence
	// of bytes by sorting map keys or struct fields according to canonical rules:
	//     - If two keys have different lengths, the shorter one sorts earlier;
	//     - If two keys have the same length, the one with the lower value in
	//       (byte-wise) lexical order sorts earlier.
	Canonical bool
	// CTAP2Canonical uses bytewise lexicographic order of map keys encodings:
	//     - If the major types are different, the one with the lower value in
	//       numerical order sorts earlier.
	//     - If two keys have different lengths, the shorter one sorts earlier;
	//     - If two keys have the same length, the one with the lower value in
	//       (byte-wise) lexical order sorts earlier.
	// Please note that when maps keys have the same data type, "canonical CBOR"
	// AND "CTAP2 canonical CBOR" render the same sort order.
	CTAP2Canonical bool
	// TimeRFC3339 causes time.Time to be encoded as string in RFC3339 format;
	// otherwise, time.Time is encoded as numerical representation of seconds
	// since January 1, 1970 UTC.
	TimeRFC3339 bool
}

// An encodeState encodes CBOR into a bytes.Buffer.
type encodeState struct {
	bytes.Buffer
	scratch [16]byte
}

// encodeStatePool caches unused encodeState objects for later reuse.
var encodeStatePool = sync.Pool{
	New: func() interface{} {
		e := new(encodeState)
		e.Grow(32) // TODO: make this configurable
		return e
	},
}

func getEncodeState() *encodeState {
	return encodeStatePool.Get().(*encodeState)
}

// putEncodeState returns e to encodeStatePool.
func putEncodeState(e *encodeState) {
	e.Reset()
	encodeStatePool.Put(e)
}

func (e *encodeState) marshal(v interface{}, opts EncOptions) error {
	_, err := encode(e, reflect.ValueOf(v), opts)
	return err
}

type encodeFunc func(e *encodeState, v reflect.Value, opts EncOptions) (int, error)

var (
	cborFalse            = []byte{0xf4}
	cborTrue             = []byte{0xf5}
	cborNil              = []byte{0xf6}
	cborNan              = []byte{0xf9, 0x7e, 0x00}
	cborPositiveInfinity = []byte{0xf9, 0x7c, 0x00}
	cborNegativeInfinity = []byte{0xf9, 0xfc, 0x00}
)

func encode(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	if !v.IsValid() {
		// v is zero value
		e.Write(cborNil)
		return 1, nil
	}
	f := getEncodeFunc(v.Type())
	if f == nil {
		return 0, &UnsupportedTypeError{v.Type()}
	}

	return f(e, v, opts)
}

func encodeBool(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	if v.Bool() {
		return e.Write(cborTrue)
	}
	return e.Write(cborFalse)
}

func encodeInt(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	i := v.Int()
	if i >= 0 {
		return encodeTypeAndAdditionalValue(e, byte(cborTypePositiveInt), uint64(i)), nil
	}
	n := v.Int()*(-1) - 1
	return encodeTypeAndAdditionalValue(e, byte(cborTypeNegativeInt), uint64(n)), nil
}

func encodeUint(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	return encodeTypeAndAdditionalValue(e, byte(cborTypePositiveInt), v.Uint()), nil
}

func encodeFloat(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	f64 := v.Float()
	if math.IsNaN(f64) {
		return e.Write(cborNan)
	}
	if math.IsInf(f64, 1) {
		return e.Write(cborPositiveInfinity)
	}
	if math.IsInf(f64, -1) {
		return e.Write(cborNegativeInfinity)
	}
	if v.Kind() == reflect.Float32 {
		f32 := v.Interface().(float32)
		e.scratch[0] = byte(cborTypePrimitives) | byte(26)
		binary.BigEndian.PutUint32(e.scratch[1:], math.Float32bits(f32))
		e.Write(e.scratch[:5])
		return 5, nil
	}
	e.scratch[0] = byte(cborTypePrimitives) | byte(27)
	binary.BigEndian.PutUint64(e.scratch[1:], math.Float64bits(f64))
	e.Write(e.scratch[:9])
	return 9, nil
}

func encodeByteString(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	if v.Kind() == reflect.Slice && v.IsNil() {
		return e.Write(cborNil)
	}
	slen := v.Len()
	if slen == 0 {
		return 1, e.WriteByte(byte(cborTypeByteString))
	}
	n1 := encodeTypeAndAdditionalValue(e, byte(cborTypeByteString), uint64(slen))
	if v.Kind() == reflect.Array {
		for i := 0; i < slen; i++ {
			e.WriteByte(byte(v.Index(i).Uint()))
		}
		return n1 + slen, nil
	}
	n2, _ := e.Write(v.Bytes())
	return n1 + n2, nil
}

func encodeString(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	return encodeStringInternal(e, v.String(), opts)
}

func encodeStringInternal(e *encodeState, s string, opts EncOptions) (int, error) {
	n1 := encodeTypeAndAdditionalValue(e, byte(cborTypeTextString), uint64(len(s)))
	n2, _ := e.WriteString(s)
	return n1 + n2, nil
}

type arrayEncoder struct {
	f encodeFunc
}

func (ae arrayEncoder) encodeArray(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	if ae.f == nil {
		return 0, &UnsupportedTypeError{v.Type()}
	}
	if v.Kind() == reflect.Slice && v.IsNil() {
		return e.Write(cborNil)
	}
	alen := v.Len()
	if alen == 0 {
		return 1, e.WriteByte(byte(cborTypeArray))
	}
	n := encodeTypeAndAdditionalValue(e, byte(cborTypeArray), uint64(alen))
	for i := 0; i < alen; i++ {
		n1, err := ae.f(e, v.Index(i), opts)
		if err != nil {
			return 0, err
		}
		n += n1
	}
	return n, nil
}

type mapEncoder struct {
	kf, ef encodeFunc
}

func (me mapEncoder) encodeMap(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	if opts.Canonical || opts.CTAP2Canonical {
		return me.encodeMapCanonical(e, v, opts)
	}
	if me.kf == nil || me.ef == nil {
		return 0, &UnsupportedTypeError{v.Type()}
	}
	if v.IsNil() {
		return e.Write(cborNil)
	}
	mlen := v.Len()
	if mlen == 0 {
		return 1, e.WriteByte(byte(cborTypeMap))
	}
	n := encodeTypeAndAdditionalValue(e, byte(cborTypeMap), uint64(mlen))
	iter := v.MapRange()
	for iter.Next() {
		n1, err := me.kf(e, iter.Key(), opts)
		if err != nil {
			return 0, err
		}
		n2, err := me.ef(e, iter.Value(), opts)
		if err != nil {
			return 0, err
		}
		n += n1 + n2
	}
	return n, nil
}

type keyValue struct {
	keyCBORData, keyValueCBORData []byte
	keyLen, keyValueLen           int
}

type pairs []keyValue

func (x pairs) Len() int {
	return len(x)
}

func (x pairs) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

type byBytewiseKeyValues struct {
	pairs
}

func (x byBytewiseKeyValues) Less(i, j int) bool {
	return bytes.Compare(x.pairs[i].keyCBORData, x.pairs[j].keyCBORData) <= 0
}

type byLengthFirstKeyValues struct {
	pairs
}

func (x byLengthFirstKeyValues) Less(i, j int) bool {
	if len(x.pairs[i].keyCBORData) != len(x.pairs[j].keyCBORData) {
		return len(x.pairs[i].keyCBORData) < len(x.pairs[j].keyCBORData)
	}
	return bytes.Compare(x.pairs[i].keyCBORData, x.pairs[j].keyCBORData) <= 0
}

var keyValuePool = sync.Pool{}

func getKeyValues(length int) []keyValue {
	v := keyValuePool.Get()
	if v == nil {
		return make([]keyValue, 0, length)
	}
	x := v.([]keyValue)
	if cap(x) < length {
		// []keyValue from the pool does not have enough capacity.
		// Return it back to the pool and create a new one.
		keyValuePool.Put(x)
		return make([]keyValue, 0, length)
	}
	return x
}

func putKeyValues(x []keyValue) {
	x = x[:0]
	keyValuePool.Put(x)
}

func (me mapEncoder) encodeMapCanonical(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	if me.kf == nil || me.ef == nil {
		return 0, &UnsupportedTypeError{v.Type()}
	}
	if v.IsNil() {
		return e.Write(cborNil)
	}
	if v.Len() == 0 {
		return 1, e.WriteByte(byte(cborTypeMap))
	}

	kve := getEncodeState()      // accumulated cbor encoded key-values
	kvs := getKeyValues(v.Len()) // for sorting keys
	iter := v.MapRange()
	for iter.Next() {
		n1, err := me.kf(kve, iter.Key(), opts)
		if err != nil {
			putEncodeState(kve)
			putKeyValues(kvs)
			return 0, err
		}
		n2, err := me.ef(kve, iter.Value(), opts)
		if err != nil {
			putEncodeState(kve)
			putKeyValues(kvs)
			return 0, err
		}
		kvs = append(kvs, keyValue{keyLen: n1, keyValueLen: n1 + n2})
	}

	b := kve.Bytes()
	for i, off := 0, 0; i < len(kvs); i++ {
		kvs[i].keyCBORData = b[off : off+kvs[i].keyLen]
		kvs[i].keyValueCBORData = b[off : off+kvs[i].keyValueLen]
		off += kvs[i].keyValueLen
	}

	if opts.CTAP2Canonical {
		sort.Sort(byBytewiseKeyValues{kvs})
	} else {
		sort.Sort(byLengthFirstKeyValues{kvs})
	}

	n := encodeTypeAndAdditionalValue(e, byte(cborTypeMap), uint64(len(kvs)))
	for i := 0; i < len(kvs); i++ {
		n1, _ := e.Write(kvs[i].keyValueCBORData)
		n += n1
	}

	putEncodeState(kve)
	putKeyValues(kvs)
	return n, nil
}

func encodeStructToArray(e *encodeState, v reflect.Value, opts EncOptions, flds fields) (int, error) {
	n := encodeTypeAndAdditionalValue(e, byte(cborTypeArray), uint64(len(flds)))
FieldLoop:
	for i := 0; i < len(flds); i++ {
		fv := v
		for k, n := range flds[i].idx {
			if k > 0 {
				if fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct {
					if fv.IsNil() {
						// Write nil for null pointer to embedded struct
						e.Write(cborNil)
						continue FieldLoop
					}
					fv = fv.Elem()
				}
			}
			fv = fv.Field(n)
		}
		n1, err := flds[i].ef(e, fv, opts)
		if err != nil {
			return 0, err
		}
		n += n1
	}
	return n, nil
}

func encodeFixedLengthStruct(e *encodeState, v reflect.Value, opts EncOptions, flds fields) (int, error) {
	n := encodeTypeAndAdditionalValue(e, byte(cborTypeMap), uint64(len(flds)))

	for i := 0; i < len(flds); i++ {
		n1, _ := e.Write(flds[i].cborName)

		fv := v.Field(flds[i].idx[0])
		n2, err := flds[i].ef(e, fv, opts)
		if err != nil {
			return 0, err
		}

		n += n1 + n2
	}

	return n, nil
}

func encodeStruct(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	structType := getEncodingStructType(v.Type())
	if structType.err != nil {
		return 0, structType.err
	}

	if structType.toArray {
		return encodeStructToArray(e, v, opts, structType.fields)
	}

	flds := structType.fields
	if opts.Canonical {
		flds = structType.lenFirstCanonicalFields
	} else if opts.CTAP2Canonical {
		flds = structType.bytewiseCanonicalFields
	}

	if !structType.hasAnonymousField && !structType.omitEmpty {
		return encodeFixedLengthStruct(e, v, opts, flds)
	}

	kve := getEncodeState() // encode key-value pairs based on struct field tag options
	kvcount := 0
FieldLoop:
	for i := 0; i < len(flds); i++ {
		fv := v
		for k, n := range flds[i].idx {
			if k > 0 {
				if fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct {
					if fv.IsNil() {
						// Null pointer to embedded struct
						continue FieldLoop
					}
					fv = fv.Elem()
				}
			}
			fv = fv.Field(n)
		}
		if flds[i].omitEmpty && isEmptyValue(fv) {
			continue
		}

		kve.Write(flds[i].cborName)

		if _, err := flds[i].ef(kve, fv, opts); err != nil {
			putEncodeState(kve)
			return 0, err
		}
		kvcount++
	}

	n1 := encodeTypeAndAdditionalValue(e, byte(cborTypeMap), uint64(kvcount))
	n2, err := e.Write(kve.Bytes())

	putEncodeState(kve)
	return n1 + n2, err
}

func encodeIntf(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	if v.IsNil() {
		return e.Write(cborNil)
	}
	return encode(e, v.Elem(), opts)
}

func encodeTime(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	t := v.Interface().(time.Time)
	if t.IsZero() {
		return e.Write(cborNil)
	} else if opts.TimeRFC3339 {
		return encodeStringInternal(e, t.Format(time.RFC3339Nano), opts)
	} else {
		t = t.UTC().Round(time.Microsecond)
		secs, nsecs := t.Unix(), uint64(t.Nanosecond())
		if nsecs == 0 {
			return encodeInt(e, reflect.ValueOf(secs), opts)
		}
		f := float64(secs) + float64(nsecs)/1e9
		return encodeFloat(e, reflect.ValueOf(f), opts)
	}
}

func encodeBinaryMarshalerType(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	m, ok := v.Interface().(encoding.BinaryMarshaler)
	if !ok {
		pv := reflect.New(v.Type())
		pv.Elem().Set(v)
		m = pv.Interface().(encoding.BinaryMarshaler)
	}
	data, err := m.MarshalBinary()
	if err != nil {
		return 0, err
	}
	n1 := encodeTypeAndAdditionalValue(e, byte(cborTypeByteString), uint64(len(data)))
	n2, _ := e.Write(data)
	return n1 + n2, nil
}

func encodeMarshalerType(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
	m, ok := v.Interface().(Marshaler)
	if !ok {
		pv := reflect.New(v.Type())
		pv.Elem().Set(v)
		m = pv.Interface().(Marshaler)
	}
	data, err := m.MarshalCBOR()
	if err != nil {
		return 0, err
	}
	return e.Write(data)
}

func encodeTypeAndAdditionalValue(e *encodeState, t byte, n uint64) int {
	if n <= 23 {
		e.WriteByte(t | byte(n))
		return 1
	} else if n <= math.MaxUint8 {
		e.scratch[0] = t | byte(24)
		e.scratch[1] = byte(n)
		e.Write(e.scratch[:2])
		return 2
	} else if n <= math.MaxUint16 {
		e.scratch[0] = t | byte(25)
		binary.BigEndian.PutUint16(e.scratch[1:], uint16(n))
		e.Write(e.scratch[:3])
		return 3
	} else if n <= math.MaxUint32 {
		e.scratch[0] = t | byte(26)
		binary.BigEndian.PutUint32(e.scratch[1:], uint32(n))
		e.Write(e.scratch[:5])
		return 5
	} else {
		e.scratch[0] = t | byte(27)
		binary.BigEndian.PutUint64(e.scratch[1:], uint64(n))
		e.Write(e.scratch[:9])
		return 9
	}
}

var (
	typeMarshaler       = reflect.TypeOf((*Marshaler)(nil)).Elem()
	typeBinaryMarshaler = reflect.TypeOf((*encoding.BinaryMarshaler)(nil)).Elem()
)

func getEncodeFuncInternal(t reflect.Type) encodeFunc {
	if t.Kind() == reflect.Ptr {
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		f := getEncodeFunc(t)
		if f == nil {
			return f
		}
		return getEncodeIndirectValueFunc(f)
	}
	if reflect.PtrTo(t).Implements(typeMarshaler) {
		return encodeMarshalerType
	}
	if reflect.PtrTo(t).Implements(typeBinaryMarshaler) {
		if t == typeTime {
			return encodeTime
		}
		return encodeBinaryMarshalerType
	}
	switch t.Kind() {
	case reflect.Bool:
		return encodeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encodeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return encodeUint
	case reflect.Float32, reflect.Float64:
		return encodeFloat
	case reflect.String:
		return encodeString
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return encodeByteString
		}
		return arrayEncoder{f: getEncodeFunc(t.Elem())}.encodeArray
	case reflect.Map:
		return mapEncoder{kf: getEncodeFunc(t.Key()), ef: getEncodeFunc(t.Elem())}.encodeMap
	case reflect.Struct:
		return encodeStruct
	case reflect.Interface:
		return encodeIntf
	default:
		return nil
	}
}

func getEncodeIndirectValueFunc(f encodeFunc) encodeFunc {
	return func(e *encodeState, v reflect.Value, opts EncOptions) (int, error) {
		for v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
		}
		if v.Kind() == reflect.Ptr && v.IsNil() {
			return e.Write(cborNil)
		}
		return f(e, v, opts)
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
