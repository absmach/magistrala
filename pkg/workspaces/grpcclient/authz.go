// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient

import (
	"context"

	grpcCommonV1 "github.com/absmach/magistrala/api/grpc/common/v1"
	grpcWorkspacesV1 "github.com/absmach/magistrala/api/grpc/workspaces/v1"
	"github.com/absmach/magistrala/pkg/grpcclient"
	pkgWorkspaces "github.com/absmach/magistrala/pkg/workspaces"
)

type authorization struct {
	workspacesSvcClient grpcWorkspacesV1.WorkspacesServiceClient
}

var _ pkgWorkspaces.Authorization = (*authorization)(nil)

func NewAuthorization(ctx context.Context, cfg grpcclient.Config) (pkgWorkspaces.Authorization, grpcWorkspacesV1.WorkspacesServiceClient, grpcclient.Handler, error) {
	workspacesClient, workspacesHandler, err := grpcclient.SetupWorkspacesClient(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	return authorization{workspacesSvcClient: workspacesClient}, workspacesClient, workspacesHandler, nil
}

func (a authorization) RetrieveStatus(ctx context.Context, id string) (pkgWorkspaces.Status, error) {
	req := grpcCommonV1.RetrieveEntityReq{
		Id: id,
	}
	res, err := a.workspacesSvcClient.RetrieveStatus(ctx, &req)
	if err != nil {
		return pkgWorkspaces.AllStatus, err
	}

	return pkgWorkspaces.Status(res.Entity.GetStatus()), nil
}
