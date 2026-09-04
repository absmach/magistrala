// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"testing"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestEncodeErrorMapsRequestErrorsToInvalidArgument(t *testing.T) {
	err := encodeError(apiutil.ErrInvalidTimeRange)

	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
