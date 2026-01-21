// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import "github.com/absmach/supermq/auth"

type issueRes struct {
	accessToken  string
	refreshToken string
	accessType   string
}

type listUserRefreshTokensRes struct {
	refreshTokens []auth.TokenInfo
}
