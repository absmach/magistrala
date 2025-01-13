// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import mgsdk "github.com/absmach/magistrala/pkg/sdk"

// Keep SDK handle in global var.
var sdk mgsdk.SDK

// SetSDK sets supermq SDK instance.
func SetSDK(s mgsdk.SDK) {
	sdk = s
}
