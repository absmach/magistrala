// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient

import (
	"context"

	grpcCommonV1 "github.com/absmach/magistrala/api/grpc/common/v1"
	grpcDomainsV1 "github.com/absmach/magistrala/api/grpc/domains/v1"
	"github.com/absmach/magistrala/domains"
	pkgDomains "github.com/absmach/magistrala/pkg/domains"
	"github.com/absmach/magistrala/pkg/grpcclient"
)

type authorization struct {
	domainsSvcClient grpcDomainsV1.DomainsServiceClient
}

var _ pkgDomains.Authorization = (*authorization)(nil)

func NewAuthorization(ctx context.Context, cfg grpcclient.Config) (pkgDomains.Authorization, grpcDomainsV1.DomainsServiceClient, grpcclient.Handler, error) {
	domainsClient, domainsHandler, err := grpcclient.SetupDomainsClient(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}

	return authorization{domainsSvcClient: domainsClient}, domainsClient, domainsHandler, nil
}

func (a authorization) RetrieveStatus(ctx context.Context, id string) (domains.Status, error) {
	req := grpcCommonV1.RetrieveEntityReq{
		Id: id,
	}
	res, err := a.domainsSvcClient.RetrieveStatus(ctx, &req)
	if err != nil {
		return domains.AllStatus, err
	}

	return domains.Status(res.Entity.GetStatus()), nil
}
