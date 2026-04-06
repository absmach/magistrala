// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/absmach/magistrala/users"

type retrieveUsersRes struct {
	users  []users.User
	total  uint64
	limit  uint64
	offset uint64
}
