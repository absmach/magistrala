// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdu

import (
	"io"

	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
)

// MaxSize is the maximum size allowed for a PDU.
const MaxSize = 4096

// Body is an abstract Protocol Data Unit (PDU) interface
// for manipulating PDUs.
type Body interface {
	// Header returns the PDU header, decoded. Header fields
	// can be updated (e.g. Seq) before re-serializing the PDU.
	Header() *Header

	// Len returns the length of the PDU binary data, in bytes.
	Len() int

	// FieldList returns a list of mandatory PDU fields for
	// encoding or decoding the PDU. The order in the list
	// dictates how PDUs are decoded and serialized.
	FieldList() pdufield.List

	// Fields return a decoded map of PDU fields. The returned
	// map can be modified before re-serializing the PDU.
	Fields() pdufield.Map

	// Fields return a decoded map of PDU TLV fields.
	TLVFields() pdutlv.Map

	// SerializeTo encodes the PDU to its binary form, including
	// the header and all fields.
	SerializeTo(w io.Writer) error
}
