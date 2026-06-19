// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
)

type Authorizer interface {
	CheckAuthz(ctx context.Context, req AuthzRequest) (AuthzResponse, error)
}

func Authorize(ctx context.Context, client Authorizer, session authn.Session, action, legacyObjectType, objectID, resourceKind string) error {
	req := AuthzRequest{
		SubjectID:  SubjectID(session),
		Action:     CapabilityName(action),
		ResourceID: resourceID(legacyObjectType, objectID),
		ObjectKind: ObjectKind(legacyObjectType, resourceKind),
		ObjectID:   objectID,
		Context: map[string]any{
			"domain_id":          session.DomainID,
			"legacy_object_type": legacyObjectType,
		},
	}
	res, err := client.CheckAuthz(ctx, req)
	if err != nil {
		return err
	}
	if !res.Allowed {
		return errors.ErrAuthorization
	}
	return nil
}

func SubjectID(session authn.Session) string {
	if session.UserID != "" {
		return session.UserID
	}
	return session.DomainUserID
}

func ObjectKind(legacyObjectType, resourceKind string) string {
	switch legacyObjectType {
	case policies.DomainType:
		return atomObjectKindTenant
	case policies.PlatformType:
		return policies.PlatformType
	case policies.ClientType:
		return atomObjectKindEntity
	case policies.GroupType:
		return atomObjectKindGroup
	case policies.ChannelType, policies.RulesType, policies.ReportsType, policies.AlarmsType:
		return atomObjectKindResource
	}
	switch resourceKind {
	case KindClient, atomKindDevice:
		return atomObjectKindEntity
	case atomKindGroup:
		return atomObjectKindGroup
	case KindChannel, KindRule, KindReport, KindAlarm:
		return atomObjectKindResource
	default:
		return resourceKind
	}
}

func resourceID(legacyObjectType, objectID string) string {
	if legacyObjectType == policies.DomainType || legacyObjectType == policies.PlatformType {
		return ""
	}
	return objectID
}
