// Copyright (c) 2019 Faye Amacker. All rights reserved.
// Use of this source code is governed by a MIT license found in the LICENSE file.

package cbor

import (
	"errors"
	"io"
	"reflect"
)

// Decoder reads and decodes CBOR values from an input stream.
type Decoder struct {
	r         io.Reader
	buf       []byte
	d         decodeState
	off       int // start of unread data in buf
	bytesRead int
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode reads the next CBOR-encoded value from its input and stores it in
// the value pointed to by v.
func (dec *Decoder) Decode(v interface{}) (err error) {
	if len(dec.buf) == dec.off {
		if n, err := dec.read(); n == 0 {
			return err
		}
	}

	dec.d.reset(dec.buf[dec.off:])
	err = dec.d.value(v)
	dec.off += dec.d.off
	dec.bytesRead += dec.d.off
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			// Need to read more data.
			if n, err := dec.read(); n == 0 {
				return err
			}
			return dec.Decode(v)
		}
		return err
	}
	return nil
}

// NumBytesRead returns the number of bytes read.
func (dec *Decoder) NumBytesRead() int {
	return dec.bytesRead
}

func (dec *Decoder) read() (int, error) {
	// Copy unread data over read data and reset off to 0.
	if dec.off > 0 {
		n := copy(dec.buf, dec.buf[dec.off:])
		dec.buf = dec.buf[:n]
		dec.off = 0
	}

	// Grow buf if needed.
	const minRead = 512
	if cap(dec.buf)-len(dec.buf) < minRead {
		newBuf := make([]byte, len(dec.buf), 2*cap(dec.buf)+minRead)
		copy(newBuf, dec.buf)
		dec.buf = newBuf
	}

	// Read from reader and reslice buf.
	n, err := dec.r.Read(dec.buf[len(dec.buf):cap(dec.buf)])
	dec.buf = dec.buf[0 : len(dec.buf)+n]
	return n, err
}

// Encoder writes CBOR values to an output stream.
type Encoder struct {
	w          io.Writer
	opts       EncOptions
	e          encodeState
	indefTypes []cborType
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer, encOpts EncOptions) *Encoder {
	return &Encoder{w: w, opts: encOpts, e: encodeState{}}
}

// Encode writes the CBOR encoding of v to the stream.
func (enc *Encoder) Encode(v interface{}) error {
	if len(enc.indefTypes) > 0 && v != nil {
		indefType := enc.indefTypes[len(enc.indefTypes)-1]
		if indefType == cborTypeTextString {
			k := reflect.TypeOf(v).Kind()
			if k != reflect.String {
				return errors.New("cbor: cannot encode item type " + k.String() + " for indefinite-length text string")
			}
		} else if indefType == cborTypeByteString {
			t := reflect.TypeOf(v)
			k := t.Kind()
			if (k != reflect.Array && k != reflect.Slice) || t.Elem().Kind() != reflect.Uint8 {
				return errors.New("cbor: cannot encode item type " + k.String() + " for indefinite-length byte string")
			}
		}
	}

	err := enc.e.marshal(v, enc.opts)
	if err == nil {
		_, err = enc.e.WriteTo(enc.w)
	}
	return err
}

// StartIndefiniteByteString starts byte string encoding of indefinite length.
// Subsequent calls of (*Encoder).Encode() encodes definite length byte strings
// ("chunks") as one continguous string until EndIndefinite is called.
func (enc *Encoder) StartIndefiniteByteString() error {
	return enc.startIndefinite(cborTypeByteString)
}

// StartIndefiniteTextString starts text string encoding of indefinite length.
// Subsequent calls of (*Encoder).Encode() encodes definite length text strings
// ("chunks") as one continguous string until EndIndefinite is called.
func (enc *Encoder) StartIndefiniteTextString() error {
	return enc.startIndefinite(cborTypeTextString)
}

// StartIndefiniteArray starts array encoding of indefinite length.
// Subsequent calls of (*Encoder).Encode() encodes elements of the array
// until EndIndefinite is called.
func (enc *Encoder) StartIndefiniteArray() error {
	return enc.startIndefinite(cborTypeArray)
}

// StartIndefiniteMap starts array encoding of indefinite length.
// Subsequent calls of (*Encoder).Encode() encodes elements of the map
// until EndIndefinite is called.
func (enc *Encoder) StartIndefiniteMap() error {
	return enc.startIndefinite(cborTypeMap)
}

// EndIndefinite closes last opened indefinite length value.
func (enc *Encoder) EndIndefinite() error {
	if len(enc.indefTypes) == 0 {
		return errors.New("cbor: cannot encode \"break\" code outside indefinite length values")
	}
	_, err := enc.w.Write([]byte{0xff})
	if err == nil {
		enc.indefTypes = enc.indefTypes[:len(enc.indefTypes)-1]
	}
	return err
}

var cborIndefHeader = map[cborType][]byte{
	cborTypeByteString: {0x5f},
	cborTypeTextString: {0x7f},
	cborTypeArray:      {0x9f},
	cborTypeMap:        {0xbf},
}

func (enc *Encoder) startIndefinite(typ cborType) error {
	_, err := enc.w.Write(cborIndefHeader[typ])
	if err == nil {
		enc.indefTypes = append(enc.indefTypes, typ)
	}
	return err
}

// RawMessage is a raw encoded CBOR value. It implements Marshaler and
// Unmarshaler interfaces and can be used to delay CBOR decoding or
// precompute a CBOR encoding.
type RawMessage []byte

// MarshalCBOR returns m as the CBOR encoding of m.
func (m RawMessage) MarshalCBOR() ([]byte, error) {
	if len(m) == 0 {
		return cborNil, nil
	}
	return m, nil
}

// UnmarshalCBOR sets *m to a copy of data.
func (m *RawMessage) UnmarshalCBOR(data []byte) error {
	if m == nil {
		return errors.New("cbor.RawMessage: UnmarshalCBOR on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}
