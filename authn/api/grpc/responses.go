// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

type identityRes struct {
	id    string
	email string
	err   error
}

type issueRes struct {
	value string
	err   error
}
