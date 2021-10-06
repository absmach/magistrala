// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutext

import (
    "golang.org/x/text/transform"
    "github.com/fiorix/go-smpp/smpp/encoding"
)

// GSM 7-bit (packed)
type GSM7Packed []byte

// Type implements the Codec interface.
func (s GSM7Packed) Type() DataCoding {
   return DefaultType
}

// Encode to GSM 7-bit (packed)
func (s GSM7Packed) Encode() []byte {
    e := encoding.GSM7(true).NewEncoder()
    es, _, err := transform.Bytes(e, s)
    if err != nil {
        return s
    }
    return es
}

// Decode from GSM 7-bit (packed)
func (s GSM7Packed) Decode() []byte {
    e := encoding.GSM7(true).NewDecoder()
    es, _, err := transform.Bytes(e, s)
    if err != nil {
        return s
    }
    return es
}
