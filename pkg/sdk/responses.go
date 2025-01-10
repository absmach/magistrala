// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import smqSDK "github.com/absmach/supermq/pkg/sdk"

// bootstrapsPage contains list of bootstrap configs in a page with proper metadata.
type BootstrapPage struct {
	Configs []BootstrapConfig `json:"configs"`
	smqSDK.PageRes
}
