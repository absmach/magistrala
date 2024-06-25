// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	domainPrefix              = "domain."
	domainCreate              = domainPrefix + "create"
	domainRetrieve            = domainPrefix + "retrieve"
	domainRetrievePermissions = domainPrefix + "retrieve_permissions"
	domainUpdate              = domainPrefix + "update"
	domainChangeStatus        = domainPrefix + "change_status"
	domainList                = domainPrefix + "list"
	domainAssign              = domainPrefix + "assign"
	domainUnassign            = domainPrefix + "unassign"
	domainUserList            = domainPrefix + "user_list"
)

var (
	_ events.Event = (*createDomainEvent)(nil)
	_ events.Event = (*retrieveDomainEvent)(nil)
	_ events.Event = (*retrieveDomainPermissionsEvent)(nil)
	_ events.Event = (*updateDomainEvent)(nil)
	_ events.Event = (*changeDomainStatusEvent)(nil)
	_ events.Event = (*listDomainsEvent)(nil)
	_ events.Event = (*assignUsersEvent)(nil)
	_ events.Event = (*unassignUsersEvent)(nil)
	_ events.Event = (*listUserDomainsEvent)(nil)
)

type createDomainEvent struct {
	auth.Domain
}

func (cde createDomainEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  domainCreate,
		"id":         cde.ID,
		"alias":      cde.Alias,
		"status":     cde.Status.String(),
		"created_at": cde.CreatedAt,
		"created_by": cde.CreatedBy,
	}

	if cde.Name != "" {
		val["name"] = cde.Name
	}
	if cde.Permission != "" {
		val["permission"] = cde.Permission
	}
	if len(cde.Tags) > 0 {
		tags := fmt.Sprintf("[%s]", strings.Join(cde.Tags, ","))
		val["tags"] = tags
	}
	if cde.Metadata != nil {
		metadata, err := json.Marshal(cde.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}

	return val, nil
}

type retrieveDomainEvent struct {
	auth.Domain
}

func (rde retrieveDomainEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  domainRetrieve,
		"id":         rde.ID,
		"alias":      rde.Alias,
		"status":     rde.Status.String(),
		"created_at": rde.CreatedAt,
	}

	if rde.Name != "" {
		val["name"] = rde.Name
	}
	if len(rde.Tags) > 0 {
		tags := fmt.Sprintf("[%s]", strings.Join(rde.Tags, ","))
		val["tags"] = tags
	}
	if rde.Metadata != nil {
		metadata, err := json.Marshal(rde.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}

	if !rde.UpdatedAt.IsZero() {
		val["updated_at"] = rde.UpdatedAt
	}
	if rde.UpdatedBy != "" {
		val["updated_by"] = rde.UpdatedBy
	}
	return val, nil
}

type retrieveDomainPermissionsEvent struct {
	domainID    string
	permissions auth.Permissions
}

func (rpe retrieveDomainPermissionsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": domainRetrievePermissions,
		"domain_id": rpe.domainID,
	}

	if rpe.permissions != nil {
		permissions, err := json.Marshal(rpe.permissions)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["permissions"] = permissions
	}

	return val, nil
}

type updateDomainEvent struct {
	auth.Domain
}

func (ude updateDomainEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  domainUpdate,
		"id":         ude.ID,
		"alias":      ude.Alias,
		"status":     ude.Status.String(),
		"created_at": ude.CreatedAt,
		"created_by": ude.CreatedBy,
		"updated_at": ude.UpdatedAt,
		"updated_by": ude.UpdatedBy,
	}

	if ude.Name != "" {
		val["name"] = ude.Name
	}
	if len(ude.Tags) > 0 {
		tags := fmt.Sprintf("[%s]", strings.Join(ude.Tags, ","))
		val["tags"] = tags
	}
	if ude.Metadata != nil {
		metadata, err := json.Marshal(ude.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}

	return val, nil
}

type changeDomainStatusEvent struct {
	domainID  string
	status    auth.Status
	updatedAt time.Time
	updatedBy string
}

func (cdse changeDomainStatusEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  domainChangeStatus,
		"id":         cdse.domainID,
		"status":     cdse.status.String(),
		"updated_at": cdse.updatedAt,
		"updated_by": cdse.updatedBy,
	}, nil
}

type listDomainsEvent struct {
	auth.Page
	total uint64
}

func (lde listDomainsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": domainList,
		"total":     lde.total,
		"offset":    lde.Offset,
		"limit":     lde.Limit,
	}

	if lde.Name != "" {
		val["name"] = lde.Name
	}
	if lde.Order != "" {
		val["order"] = lde.Order
	}
	if lde.Dir != "" {
		val["dir"] = lde.Dir
	}
	if lde.Metadata != nil {
		metadata, err := json.Marshal(lde.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lde.Tag != "" {
		val["tag"] = lde.Tag
	}
	if lde.Permission != "" {
		val["permission"] = lde.Permission
	}
	if lde.Status.String() != "" {
		val["status"] = lde.Status.String()
	}
	if lde.ID != "" {
		val["id"] = lde.ID
	}
	if len(lde.IDs) > 0 {
		val["ids"] = lde.IDs
	}
	if lde.Identity != "" {
		val["identity"] = lde.Identity
	}
	if lde.SubjectID != "" {
		val["subject_id"] = lde.SubjectID
	}

	return val, nil
}

type assignUsersEvent struct {
	userIDs  []string
	domainID string
	relation string
}

func (ase assignUsersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": domainAssign,
		"user_ids":  ase.userIDs,
		"domain_id": ase.domainID,
		"relation":  ase.relation,
	}

	return val, nil
}

type unassignUsersEvent struct {
	userIDs  []string
	domainID string
}

func (use unassignUsersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": domainUnassign,
		"user_ids":  use.userIDs,
		"domain_id": use.domainID,
	}

	return val, nil
}

type listUserDomainsEvent struct {
	auth.Page
	userID string
}

func (lde listUserDomainsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": domainUserList,
		"total":     lde.Total,
		"offset":    lde.Offset,
		"limit":     lde.Limit,
		"user_id":   lde.userID,
	}

	if lde.Name != "" {
		val["name"] = lde.Name
	}
	if lde.Order != "" {
		val["order"] = lde.Order
	}
	if lde.Dir != "" {
		val["dir"] = lde.Dir
	}
	if lde.Metadata != nil {
		metadata, err := json.Marshal(lde.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lde.Tag != "" {
		val["tag"] = lde.Tag
	}
	if lde.Permission != "" {
		val["permission"] = lde.Permission
	}
	if lde.Status.String() != "" {
		val["status"] = lde.Status.String()
	}
	if lde.ID != "" {
		val["id"] = lde.ID
	}
	if len(lde.IDs) > 0 {
		val["ids"] = lde.IDs
	}
	if lde.Identity != "" {
		val["identity"] = lde.Identity
	}
	if lde.SubjectID != "" {
		val["subject_id"] = lde.SubjectID
	}

	return val, nil
}
