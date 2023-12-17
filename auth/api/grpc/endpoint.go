// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/go-kit/kit/endpoint"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type:   req.keyType,
			User:   req.userID,
			Domain: req.domainID,
		}
		tkn, err := svc.Issue(ctx, "", key)
		if err != nil {
			return issueRes{}, err
		}
		ret := issueRes{
			accessToken:  tkn.AccessToken,
			refreshToken: tkn.RefreshToken,
			accessType:   tkn.AccessType,
		}
		return ret, nil
	}
}

func refreshEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(refreshReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{Domain: req.domainID, Type: auth.RefreshKey}
		tkn, err := svc.Issue(ctx, req.refreshToken, key)
		if err != nil {
			return issueRes{}, err
		}
		ret := issueRes{
			accessToken:  tkn.AccessToken,
			refreshToken: tkn.RefreshToken,
			accessType:   tkn.AccessType,
		}
		return ret, nil
	}
}

func identifyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}

		key, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identityRes{}, err
		}

		return identityRes{id: key.Subject, userID: key.User, domainID: key.Domain}, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}
		err := svc.Authorize(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			SubjectKind: req.SubjectKind,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return authorizeRes{authorized: false}, err
		}
		return authorizeRes{authorized: true}, nil
	}
}

func addPolicyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policyReq)
		if err := req.validate(); err != nil {
			return addPolicyRes{}, err
		}

		err := svc.AddPolicy(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			SubjectKind: req.SubjectKind,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			ObjectKind:  req.ObjectKind,
			Object:      req.Object,
		})
		if err != nil {
			return addPolicyRes{}, err
		}
		return addPolicyRes{authorized: true}, err
	}
}

func addPoliciesEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqs := request.(policiesReq)
		if err := reqs.validate(); err != nil {
			return addPoliciesRes{}, err
		}

		prs := []auth.PolicyReq{}

		for _, req := range reqs {
			prs = append(prs, auth.PolicyReq{
				Domain:      req.Domain,
				SubjectType: req.SubjectType,
				SubjectKind: req.SubjectKind,
				Subject:     req.Subject,
				Relation:    req.Relation,
				Permission:  req.Permission,
				ObjectType:  req.ObjectType,
				ObjectKind:  req.ObjectKind,
				Object:      req.Object,
			})
		}

		if err := svc.AddPolicies(ctx, prs); err != nil {
			return addPoliciesRes{}, err
		}
		return addPoliciesRes{authorized: true}, nil
	}
}

func deletePolicyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policyReq)
		if err := req.validate(); err != nil {
			return deletePolicyRes{}, err
		}

		err := svc.DeletePolicy(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectKind: req.SubjectKind,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			ObjectKind:  req.ObjectKind,
			Object:      req.Object,
		})
		if err != nil {
			return deletePolicyRes{}, err
		}
		return deletePolicyRes{deleted: true}, nil
	}
}

func deletePoliciesEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		reqs := request.(policiesReq)
		if err := reqs.validate(); err != nil {
			return deletePoliciesRes{}, err
		}

		prs := []auth.PolicyReq{}

		for _, req := range reqs {
			prs = append(prs, auth.PolicyReq{
				Domain:      req.Domain,
				SubjectType: req.SubjectType,
				SubjectKind: req.SubjectKind,
				Subject:     req.Subject,
				Relation:    req.Relation,
				Permission:  req.Permission,
				ObjectType:  req.ObjectType,
				ObjectKind:  req.ObjectKind,
				Object:      req.Object,
			})
		}

		if err := svc.DeletePolicies(ctx, prs); err != nil {
			return deletePoliciesRes{}, err
		}
		return deletePoliciesRes{deleted: true}, nil
	}
}

func listObjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listObjectsReq)

		page, err := svc.ListObjects(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		}, req.NextPageToken, req.Limit)
		if err != nil {
			return listObjectsRes{}, err
		}
		return listObjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func listAllObjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listObjectsReq)

		page, err := svc.ListAllObjects(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return listObjectsRes{}, err
		}
		return listObjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func countObjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(countObjectsReq)

		count, err := svc.CountObjects(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return countObjectsRes{}, err
		}
		return countObjectsRes{count: count}, nil
	}
}

func listSubjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSubjectsReq)

		page, err := svc.ListSubjects(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		}, req.NextPageToken, req.Limit)
		if err != nil {
			return listSubjectsRes{}, err
		}
		return listSubjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func listAllSubjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSubjectsReq)

		page, err := svc.ListAllSubjects(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return listSubjectsRes{}, err
		}
		return listSubjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func countSubjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(countSubjectsReq)

		count, err := svc.CountSubjects(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return countSubjectsRes{}, err
		}
		return countSubjectsRes{count: count}, nil
	}
}

func listPermissionsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listPermissionsReq)
		permissions, err := svc.ListPermissions(ctx, auth.PolicyReq{
			SubjectType:     req.SubjectType,
			SubjectRelation: req.SubjectRelation,
			Subject:         req.Subject,
			Object:          req.Object,
			ObjectType:      req.ObjectType,
		}, req.FilterPermissions)
		if err != nil {
			return listPermissionsRes{}, err
		}
		return listPermissionsRes{
			SubjectType:     req.SubjectType,
			SubjectRelation: req.SubjectRelation,
			Subject:         req.Subject,
			Object:          req.Object,
			ObjectType:      req.ObjectType,
			Permissions:     permissions,
		}, nil
	}
}
