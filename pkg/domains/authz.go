// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"

	"github.com/absmach/magistrala/domains"
)

type Authorization interface {
	RetrieveStatus(ctx context.Context, id string) (domains.Status, error)
}
