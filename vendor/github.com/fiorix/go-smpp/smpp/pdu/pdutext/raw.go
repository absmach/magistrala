// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutext

// Raw text codec, no encoding.
type Raw []byte

// Type implements the Codec interface.
func (s Raw) Type() DataCoding {
	return DefaultType
}

// Encode raw text.
func (s Raw) Encode() []byte {
	return s
}

// Decode raw text.
func (s Raw) Decode() []byte {
	return s
}
