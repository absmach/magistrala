// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"strings"

	smqauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
)

type AuthorizationCompat struct {
	Client Authorizer
}

func NewAuthorizationCompat(client Authorizer) AuthorizationCompat {
	return AuthorizationCompat{Client: client}
}

func (a AuthorizationCompat) Authorize(ctx context.Context, pr smqauthz.PolicyReq, _ *smqauthz.PATReq) error {
	subjectID := pr.Subject
	if subjectID == "" {
		return errors.ErrAuthentication
	}
	objectKind := ObjectKind(pr.ObjectType, legacyResourceKind(pr.ObjectKind, pr.ObjectType))
	res, err := a.Client.CheckAuthz(ctx, AuthzRequest{
		SubjectID:  subjectID,
		Action:     CapabilityName(pr.Permission),
		ResourceID: resourceID(pr.ObjectType, pr.Object),
		ObjectKind: objectKind,
		ObjectID:   pr.Object,
		Context: map[string]any{
			"domain_id":           pr.Domain,
			"legacy_object_kind":  pr.ObjectKind,
			"legacy_object_type":  pr.ObjectType,
			"legacy_permission":   pr.Permission,
			"legacy_relation":     pr.Relation,
			"legacy_subject_kind": pr.SubjectKind,
			"legacy_subject_type": pr.SubjectType,
		},
	})
	if err != nil {
		return err
	}
	if !res.Allowed {
		return errors.ErrAuthorization
	}
	return nil
}

func CapabilityName(action string) string {
	normalized := strings.ToLower(strings.TrimSpace(action))
	switch {
	case normalized == policies.AdminPermission,
		normalized == "admin_permission",
		strings.Contains(normalized, "manage_role"):
		return atomActionManage
	case normalized == policies.ViewPermission,
		normalized == atomActionRead,
		strings.Contains(normalized, "read"),
		strings.Contains(normalized, "view"):
		return atomActionRead
	case normalized == policies.CreatePermission,
		normalized == "write",
		strings.Contains(normalized, "create"),
		strings.Contains(normalized, "update"),
		strings.Contains(normalized, "edit"),
		strings.Contains(normalized, "enable"),
		strings.Contains(normalized, "disable"),
		strings.Contains(normalized, "assign"),
		strings.Contains(normalized, "acknowledge"),
		strings.Contains(normalized, "resolve"):
		return atomActionWrite
	case normalized == policies.DeletePermission,
		strings.Contains(normalized, "delete"),
		strings.Contains(normalized, "remove"):
		return atomActionDelete
	case normalized == policies.PublishPermission:
		return atomActionPublish
	case normalized == policies.SubscribePermission:
		return atomActionSubscribe
	case normalized == "generate", normalized == "execute":
		return atomActionExecute
	case normalized == "list":
		return "list"
	default:
		return normalized
	}
}

func legacyResourceKind(objectKind, objectType string) string {
	switch objectKind {
	case policies.ChannelsKind, policies.NewChannelKind:
		return KindChannel
	case policies.ClientsKind, policies.NewClientKind:
		return "client"
	case policies.GroupsKind, policies.NewGroupKind:
		return atomObjectKindGroup
	case policies.DomainsKind:
		return atomObjectKindTenant
	default:
		switch objectType {
		case policies.ChannelType:
			return KindChannel
		case policies.ClientType:
			return "client"
		case policies.GroupType:
			return atomObjectKindGroup
		case policies.DomainType:
			return atomObjectKindTenant
		case policies.RulesType:
			return KindRule
		case policies.ReportsType:
			return KindReport
		case policies.AlarmsType:
			return KindAlarm
		default:
			return objectType
		}
	}
}
