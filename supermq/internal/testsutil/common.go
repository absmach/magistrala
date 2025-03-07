// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package testsutil

import (
	"fmt"
	"testing"

	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func GenerateUUID(t *testing.T) string {
	idProvider := uuid.New()
	ulid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	return ulid
}
