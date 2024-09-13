// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policy"
	"github.com/absmach/magistrala/users"
	"github.com/go-kit/kit/endpoint"
)

var errIssueToken = errors.New("failed to issue token")

func registrationEndpoint(svc users.Service, authClient auth.AuthClient, selfRegister bool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session := auth.Session{}

		if !selfRegister {
			session, err := identify(ctx, authClient, req.token)
			if err != nil {
				return nil, err
			}
			if err := checkSuperAdmin(ctx, authClient, session.UserID); err != nil {
				return nil, err
			}
		}

		client, err := svc.RegisterClient(ctx, session, req.client, selfRegister)
		if err != nil {
			return nil, err
		}

		return createClientRes{
			Client:  client,
			created: true,
		}, nil
	}
}

func viewClientEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}

		client, err := svc.ViewClient(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return viewClientRes{Client: client}, nil
	}
}

func viewProfileEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		client, err := svc.ViewProfile(ctx, session)
		if err != nil {
			return nil, err
		}

		return viewClientRes{Client: client}, nil
	}
}

func listClientsEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}

		pm := mgclients.Page{
			Status:   req.status,
			Offset:   req.offset,
			Limit:    req.limit,
			Name:     req.name,
			Tag:      req.tag,
			Metadata: req.metadata,
			Identity: req.identity,
			Order:    req.order,
			Dir:      req.dir,
			Id:       req.id,
		}

		page, err := svc.ListClients(ctx, session, pm)
		if err != nil {
			return nil, err
		}

		res := clientsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Clients: []viewClientRes{},
		}
		for _, client := range page.Clients {
			res.Clients = append(res.Clients, viewClientRes{Client: client})
		}

		return res, nil
	}
}

func searchClientsEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		_, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}

		pm := mgclients.Page{
			Offset: req.Offset,
			Limit:  req.Limit,
			Name:   req.Name,
			Id:     req.Id,
			Order:  req.Order,
			Dir:    req.Dir,
		}
		page, err := svc.SearchUsers(ctx, pm)
		if err != nil {
			return nil, err
		}

		res := clientsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Clients: []viewClientRes{},
		}
		for _, client := range page.Clients {
			res.Clients = append(res.Clients, viewClientRes{Client: client})
		}

		return res, nil
	}
}

func listMembersByGroupEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		req.objectKind = "groups"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err = authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, mgauth.SwitchToPermission(req.Page.Permission), policy.GroupType, req.objectID); err != nil {
			return nil, err
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildClientsResponse(page), nil
	}
}

func listMembersByChannelEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		// In spiceDB schema, using the same 'group' type for both channels and groups, rather than having a separate type for channels.
		req.objectKind = "groups"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, mgauth.SwitchToPermission(req.Page.Permission), policy.GroupType, req.objectID); err != nil {
			return nil, err
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildClientsResponse(page), nil
	}
}

func listMembersByThingEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		req.objectKind = "things"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, req.Page.Permission, policy.ThingType, req.objectID); err != nil {
			return nil, err
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildClientsResponse(page), nil
	}
}

func listMembersByDomainEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByObjectReq)
		req.objectKind = "domains"
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, mgauth.SwitchToPermission(req.Page.Permission), policy.DomainType, req.objectID); err != nil {
			return nil, err
		}

		page, err := svc.ListMembers(ctx, session, req.objectKind, req.objectID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildClientsResponse(page), nil
	}
}

func updateClientEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}

		client := mgclients.Client{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		client, err = svc.UpdateClient(ctx, session, client)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientTagsEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}

		client := mgclients.Client{
			ID:   req.id,
			Tags: req.Tags,
		}

		client, err = svc.UpdateClientTags(ctx, session, client)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientIdentityEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientIdentityReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}

		client, err := svc.UpdateClientIdentity(ctx, session, req.id, req.Identity)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

// Password reset request endpoint.
// When successful password reset link is generated.
// Link is generated using MG_TOKEN_RESET_ENDPOINT env.
// and value from Referer header for host.
// {Referer}+{MG_TOKEN_RESET_ENDPOINT}+{token=TOKEN}
// http://magistrala.com/reset-request?token=xxxxxxxxxxx.
// Email with a link is being sent to the user.
// When user clicks on a link it should get the ui with form to
// enter new password, when form is submitted token and new password
// must be sent as PUT request to 'password/reset' passwordResetEndpoint.
func passwordResetRequestEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwResetReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.GenerateResetToken(ctx, req.Email, req.Host)
		if err != nil {
			return nil, err
		}
		token, err := authClient.Issue(ctx, &magistrala.IssueReq{
			UserId: client.ID,
			Type:   uint32(mgauth.RecoveryKey),
		})
		if err != nil {
			return nil, errors.Wrap(errIssueToken, err)
		}
		err = svc.SendPasswordReset(ctx, req.Host, req.Email, client.Name, token.AccessToken)
		if err != nil {
			return nil, err
		}

		return passwResetReqRes{Msg: MailSent}, nil
	}
}

// This is endpoint that actually sets new password in password reset flow.
// When user clicks on a link in email finally ends on this endpoint as explained in
// the comment above.
func passwordResetEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetTokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.Token)
		if err != nil {
			return nil, err
		}
		if err := svc.ResetSecret(ctx, session, req.Password); err != nil {
			return nil, err
		}

		return passwChangeRes{}, nil
	}
}

func updateClientSecretEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientSecretReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}

		client, err := svc.UpdateClientSecret(ctx, session, req.OldSecret, req.NewSecret)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientRoleEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client := mgclients.Client{
			ID:   req.id,
			Role: req.role,
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}
		if err := authorize(ctx, authClient, "", policy.UserType, policy.UsersKind, client.ID, policy.MembershipPermission, policy.PlatformType, policy.MagistralaObject); err != nil {
			return nil, err
		}

		client, err = svc.UpdateClientRole(ctx, session, client)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func issueTokenEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(loginClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.IssueToken(ctx, req.Identity, req.Secret, req.DomainID)
		if err != nil {
			return nil, err
		}

		token, err := authClient.Issue(ctx, &magistrala.IssueReq{
			UserId:   client.ID,
			DomainId: &client.Domain,
			Type:     uint32(mgauth.AccessKey),
		})
		if err != nil {
			return nil, errors.Wrap(errIssueToken, err)
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func refreshTokenEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(tokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.RefreshToken)
		if err != nil {
			return nil, err
		}
		client, err := svc.RefreshToken(ctx, session, req.DomainID)
		if err != nil {
			return nil, err
		}

		token, err := authClient.Refresh(ctx, &magistrala.RefreshReq{
			RefreshToken: req.RefreshToken,
			DomainId:     &client.Domain,
		})
		if err != nil {
			return nil, errors.Wrap(errIssueToken, err)
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func enableClientEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}
		client, err := svc.EnableClient(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeClientStatusClientRes{Client: client}, nil
	}
}

func disableClientEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}
		client, err := svc.DisableClient(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeClientStatusClientRes{Client: client}, nil
	}
}

func deleteClientEndpoint(svc users.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if err := checkSuperAdmin(ctx, authClient, session.UserID); err == nil {
			session.SuperAdmin = true
		}
		if err := svc.DeleteClient(ctx, session, req.id); err != nil {
			return nil, err
		}

		return deleteClientRes{true}, nil
	}
}

func buildClientsResponse(cp mgclients.MembersPage) clientsPageRes {
	res := clientsPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
		},
		Clients: []viewClientRes{},
	}

	for _, client := range cp.Members {
		res.Clients = append(res.Clients, viewClientRes{Client: client})
	}

	return res
}

func identify(ctx context.Context, authClient auth.AuthClient, token string) (auth.Session, error) {
	resp, err := authClient.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return auth.Session{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return auth.Session{
		DomainUserID: resp.GetId(),
		UserID:       resp.GetUserId(),
		DomainID:     resp.GetDomainId(),
	}, nil
}

func authorize(ctx context.Context, authClient auth.AuthClient, domainID string, subjectType, subjectKind, subject, permission, objectType, objectID string) error {
	res, err := authClient.Authorize(ctx, &magistrala.AuthorizeReq{
		Domain:      domainID,
		SubjectType: subjectType,
		SubjectKind: subjectKind,
		Subject:     subject,
		Permission:  permission,
		ObjectType:  objectType,
		Object:      objectID,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.Authorized {
		return svcerr.ErrAuthorization
	}
	return nil
}

func checkSuperAdmin(ctx context.Context, authClient auth.AuthClient, adminID string) error {
	if _, err := authClient.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: policy.UserType,
		SubjectKind: policy.UsersKind,
		Subject:     adminID,
		Permission:  policy.AdminPermission,
		ObjectType:  policy.PlatformType,
		Object:      policy.MagistralaObject,
	}); err != nil {
		return err
	}
	return nil
}
