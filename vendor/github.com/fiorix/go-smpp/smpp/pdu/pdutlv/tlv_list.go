// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutlv

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// DecodeTLV scans the given byte slice to build a Map from binary data.
func DecodeTLV(r *bytes.Buffer) (Map, error) {
	t := make(Map)
	for r.Len() >= 4 {
		b := r.Next(4)
		ft := Tag(binary.BigEndian.Uint16(b[0:2]))
		fl := binary.BigEndian.Uint16(b[2:4])
		if r.Len() < int(fl) {
			return nil, fmt.Errorf("not enough data for tag %s: want %d, have %d",
				ft.Hex(), fl, r.Len())
		}
		b = r.Next(int(fl))
		t[ft] = &Field{
			Tag:  ft,
			Data: b,
		}
	}
	return t, nil
}
