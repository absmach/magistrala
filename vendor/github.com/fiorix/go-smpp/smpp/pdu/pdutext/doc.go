// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package pdutext provides text conversion for PDU fields.
//
// See 2.2.2 from http://opensmpp.org/specs/smppv34_gsmumts_ig_v10.pdf
// for details.
//
// pdutext supports Latin1 (0x03) and UCS2 (0x08).
//
// Latin1 encoding is Windows-1252 (CP1252) for now, not ISO-8859-1.
// http://www.i18nqa.com/debug/table-iso8859-1-vs-windows-1252.html
//
// UCS2 is UTF-16-BE. Here be dragons.
//
// TODO(fiorix): Fix this.
package pdutext
