// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authsvc

import (
	"context"

	grpcAuthV1 "github.com/absmach/magistrala/api/grpc/auth/v1"
	"github.com/absmach/magistrala/auth/api/grpc/auth"
	"github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/policies"
	pkgWorkspaces "github.com/absmach/magistrala/pkg/workspaces"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

type authorization struct {
	authSvcClient grpcAuthV1.AuthServiceClient
	workspaces    pkgWorkspaces.Authorization
}

var _ authz.Authorization = (*authorization)(nil)

func NewAuthorization(ctx context.Context, cfg grpcclient.Config, workspacesAuthz pkgWorkspaces.Authorization) (authz.Authorization, grpcclient.Handler, error) {
	client, err := grpcclient.NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "auth",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, grpcclient.ErrSvcNotServing
	}

	authSvcClient := auth.NewAuthClient(client.Connection(), cfg.Timeout)
	return authorization{
		authSvcClient: authSvcClient,
		workspaces:    workspacesAuthz,
	}, client, nil
}

func (a authorization) Authorize(ctx context.Context, pr authz.PolicyReq, pat *authz.PATReq) error {
	if pr.SubjectType == policies.UserType && (pr.ObjectType == policies.GroupType || pr.ObjectType == policies.ClientType || pr.ObjectType == policies.WorkspaceType) {
		workspaceID := pr.Workspace
		if workspaceID == "" {
			if pr.ObjectType != policies.WorkspaceType {
				return svcerr.ErrWorkspaceAuthorization
			}
			workspaceID = pr.Object
		}
		if err := a.checkWorkspace(ctx, pr.SubjectType, pr.Subject, workspaceID); err != nil {
			return errors.Wrap(svcerr.ErrWorkspaceAuthorization, err)
		}
	}

	req := grpcAuthV1.AuthZReq{
		PolicyReq: &grpcAuthV1.PolicyReq{
			Workspace:       pr.Workspace,
			SubjectType:     pr.SubjectType,
			SubjectKind:     pr.SubjectKind,
			SubjectRelation: pr.SubjectRelation,
			Subject:         pr.Subject,
			Relation:        pr.Relation,
			Permission:      pr.Permission,
			Object:          pr.Object,
			ObjectType:      pr.ObjectType,
		},
	}

	if pat != nil {
		req.PatReq = &grpcAuthV1.PATReq{
			PatId:      pat.PatID,
			Workspace:  pat.Workspace,
			Operation:  pat.Operation,
			UserId:     pat.UserID,
			EntityId:   pat.EntityID,
			EntityType: pat.EntityType,
		}
	}

	res, err := a.authSvcClient.Authorize(ctx, &req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}

func (a authorization) checkWorkspace(ctx context.Context, subjectType, subject, workspaceID string) error {
	status, err := a.workspaces.RetrieveStatus(ctx, workspaceID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	switch status {
	case pkgWorkspaces.FreezeStatus:
		_, err := a.authSvcClient.Authorize(ctx, &grpcAuthV1.AuthZReq{
			PolicyReq: &grpcAuthV1.PolicyReq{
				Subject:     subject,
				SubjectType: subjectType,
				Permission:  policies.AdminPermission,
				Object:      policies.MagistralaObject,
				ObjectType:  policies.PlatformType,
			},
		})

		return err
	case pkgWorkspaces.DisabledStatus:
		_, err := a.authSvcClient.Authorize(ctx, &grpcAuthV1.AuthZReq{
			PolicyReq: &grpcAuthV1.PolicyReq{
				Subject:     subject,
				SubjectType: subjectType,
				Permission:  policies.AdminPermission,
				Object:      workspaceID,
				ObjectType:  policies.WorkspaceType,
			},
		})

		return err
	case pkgWorkspaces.EnabledStatus:
		_, err := a.authSvcClient.Authorize(ctx, &grpcAuthV1.AuthZReq{
			PolicyReq: &grpcAuthV1.PolicyReq{
				Subject:     subject,
				SubjectType: subjectType,
				Permission:  policies.MembershipPermission,
				Object:      workspaceID,
				ObjectType:  policies.WorkspaceType,
			},
		})

		return err
	default:
		return svcerr.ErrInvalidStatus
	}
}
