// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

func readString(data map[string]any, key string) string {
	val, _ := data[key].(string)
	return val
}
