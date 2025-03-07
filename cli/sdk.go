// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import smqsdk "github.com/absmach/supermq/pkg/sdk"

// Keep SDK handle in global var.
var sdk smqsdk.SDK

// SetSDK sets supermq SDK instance.
func SetSDK(s smqsdk.SDK) {
	sdk = s
}
