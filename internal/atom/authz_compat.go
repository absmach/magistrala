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
		return "manage"
	case normalized == policies.ViewPermission,
		normalized == "read",
		strings.Contains(normalized, "read"),
		strings.Contains(normalized, "view"):
		return "read"
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
		return "write"
	case normalized == policies.DeletePermission,
		strings.Contains(normalized, "delete"),
		strings.Contains(normalized, "remove"):
		return "delete"
	case normalized == policies.PublishPermission:
		return "publish"
	case normalized == policies.SubscribePermission:
		return "subscribe"
	case normalized == "generate", normalized == "execute":
		return "execute"
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
		return "group"
	case policies.DomainsKind:
		return "tenant"
	default:
		switch objectType {
		case policies.ChannelType:
			return KindChannel
		case policies.ClientType:
			return "client"
		case policies.GroupType:
			return "group"
		case policies.DomainType:
			return "tenant"
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
