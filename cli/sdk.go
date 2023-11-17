// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"

// Keep SDK handle in global var.
var sdk mgxsdk.SDK

// SetSDK sets magistrala SDK instance.
func SetSDK(s mgxsdk.SDK) {
	sdk = s
}
