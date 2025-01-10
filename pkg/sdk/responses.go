// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

type PageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

// bootstrapsPage contains list of bootstrap configs in a page with proper metadata.
type BootstrapPage struct {
	Configs []BootstrapConfig `json:"configs"`
	PageRes
}
