// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdufield

import (
	"fmt"

	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

// Map is a collection of PDU field data indexed by name.
type Map map[Name]Body

// Set updates the PDU map with the given key and value, and
// returns error if the value cannot be converted to type Data.
//
// This is a shortcut for m[k] = New(k, v) converting v properly.
//
// If k is ShortMessage and v is of type pdutext.Codec, text is
// encoded and data_coding PDU and sm_length PDUs are set.
func (m Map) Set(k Name, v interface{}) error {
	switch v.(type) {
	case nil:
		m[k] = New(k, nil) // use default value
	case uint8:
		m[k] = New(k, []byte{v.(uint8)})
	case int:
		m[k] = New(k, []byte{uint8(v.(int))})
	case string:
		m[k] = New(k, []byte(v.(string)))
	case []byte:
		m[k] = New(k, []byte(v.([]byte)))
	case DeliverySetting:
		m[k] = New(k, []byte{uint8(v.(DeliverySetting))})
	case Body:
		m[k] = v.(Body)
	case pdutext.Codec:
		c := v.(pdutext.Codec)
		m[k] = New(k, c.Encode())
		if k == ShortMessage {
			m[DataCoding] = &Fixed{Data: uint8(c.Type())}
		}
	default:
		return fmt.Errorf("unsupported field data: %#v", v)
	}
	if k == ShortMessage {
		m[SMLength] = &Fixed{Data: uint8(m[k].Len())}
	}
	return nil
}
