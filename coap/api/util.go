//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import "strings"

func authKey(opt interface{}) (string, error) {
	val, ok := opt.(string)
	if !ok {
		return "", errBadRequest
	}

	arr := strings.Split(val, "=")
	if len(arr) != 2 || strings.ToLower(arr[0]) != "authorization" {
		return "", errBadOption
	}

	return arr[1], nil
}
