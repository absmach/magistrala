// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package readersclient provides the public gRPC client used to read messages
// from Community-owned reader services.
package readersclient

import (
	"context"
	"fmt"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ grpcReadersV1.ReadersServiceClient = (*client)(nil)

type client struct {
	readers grpcReadersV1.ReadersServiceClient
	timeout time.Duration
}

// New returns a reader gRPC client with a per-request timeout.
func New(conn *grpc.ClientConn, timeout time.Duration) grpcReadersV1.ReadersServiceClient {
	return &client{
		readers: grpcReadersV1.NewReadersServiceClient(conn),
		timeout: timeout,
	}
}

func (c *client) ReadMessages(ctx context.Context, req *grpcReadersV1.ReadMessagesReq, opts ...grpc.CallOption) (*grpcReadersV1.ReadMessagesRes, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.readers.ReadMessages(ctx, req, opts...)
	if err != nil {
		return &grpcReadersV1.ReadMessagesRes{}, decodeError(err)
	}
	return res, nil
}

func decodeError(err error) error {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unauthenticated:
			return errors.Wrap(svcerr.ErrAuthentication, errors.New(st.Message()))
		case codes.PermissionDenied:
			return errors.Wrap(svcerr.ErrAuthorization, errors.New(st.Message()))
		case codes.InvalidArgument, codes.FailedPrecondition:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.NotFound:
			return errors.Wrap(svcerr.ErrNotFound, errors.New(st.Message()))
		case codes.AlreadyExists:
			return errors.Wrap(svcerr.ErrConflict, errors.New(st.Message()))
		case codes.OK:
			if msg := st.Message(); msg != "" {
				return errors.Wrap(errors.ErrUnidentified, errors.New(msg))
			}
			return nil
		default:
			return errors.Wrap(fmt.Errorf("unexpected gRPC status: %s (status code:%v)", st.Code().String(), st.Code()), errors.New(st.Message()))
		}
	}
	return err
}
