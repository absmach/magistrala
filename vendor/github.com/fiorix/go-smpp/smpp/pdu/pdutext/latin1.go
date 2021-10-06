// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutext

import (
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Latin1 text codec.
type Latin1 []byte

// Type implements the Codec interface.
func (s Latin1) Type() DataCoding {
	return Latin1Type
}

// Encode to Latin1.
func (s Latin1) Encode() []byte {
	e := charmap.Windows1252.NewEncoder()
	es, _, err := transform.Bytes(e, s)
	if err != nil {
		return s
	}
	return es
}

// Decode from Latin1.
func (s Latin1) Decode() []byte {
	e := charmap.Windows1252.NewDecoder()
	es, _, err := transform.Bytes(e, s)
	if err != nil {
		return s
	}
	return es
}
