// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import "strings"

var serialReplacer = strings.NewReplacer(":", "", " ", "")

// NormalizeSerialNumber normalizes a serial number to use colon-separated hex format.
func NormalizeSerialNumber(serial string) string {
	if len(serial) < 2 {
		return serialReplacer.Replace(serial)
	}
	cleaned := serialReplacer.Replace(serial)
	cleaned = strings.ToLower(cleaned)
	if len(cleaned)%2 != 0 {
		cleaned = "0" + cleaned
	}

	capacity := len(cleaned) + (len(cleaned)/2 - 1)
	var result strings.Builder
	result.Grow(capacity)
	for i := 0; i < len(cleaned); i += 2 {
		if i > 0 {
			result.WriteString(":")
		}
		result.WriteString(cleaned[i : i+2])
	}

	return result.String()
}
