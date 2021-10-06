// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutlv

import (
	"fmt"
)

// Map is a collection of PDU TLV field data indexed by tag.
type Map map[Tag]Body

// Set updates the PDU map with the given tag and value, and
// returns error if the value cannot be converted to type Data.
//
// This is a shortcut for m[t] = NewTLV(t, v) converting v properly.
func (m Map) Set(t Tag, v interface{}) error {
	switch v.(type) {
	case nil:
		m[t] = NewTLV(t, nil) // use default value
	case uint8:
		m[t] = NewTLV(t, []byte{v.(uint8)})
	case int:
		m[t] = NewTLV(t, []byte{uint8(v.(int))})
	case string:
		m[t] = NewTLV(t, []byte(v.(string)))
	case String:
		m[t] = NewTLV(t, []byte(v.(String)))
	case CString:
		value := []byte(v.(CString))
		if len(value) == 0 || value[len(value)-1] != 0x00 {
			value = append(value, 0x00)
		}
		m[t] = NewTLV(t, value)
	case []byte:
		m[t] = NewTLV(t, []byte(v.([]byte)))
	case Body:
		m[t] = v.(Body)
	default:
		return fmt.Errorf("unsupported Tag-Length-Value field data: %#v", v)
	}
	return nil
}
