// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutext

import (
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// ISO88595 text codec.
type ISO88595 []byte

// Type implements the Codec interface.
func (s ISO88595) Type() DataCoding {
	return ISO88595Type
}

// Encode to ISO88595.
func (s ISO88595) Encode() []byte {
	e := charmap.ISO8859_5.NewEncoder()
	es, _, err := transform.Bytes(e, s)
	if err != nil {
		return s
	}
	return es
}

// Decode from ISO88595.
func (s ISO88595) Decode() []byte {
	e := charmap.ISO8859_5.NewDecoder()
	es, _, err := transform.Bytes(e, s)
	if err != nil {
		return s
	}
	return es
}
