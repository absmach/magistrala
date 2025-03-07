// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"

	"github.com/absmach/supermq/domains"
)

type Authorization interface {
	RetrieveEntity(ctx context.Context, id string) (domains.Domain, error)
}
