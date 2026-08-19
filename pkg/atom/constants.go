// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

const (
	atomActionRead      = "read"
	atomActionWrite     = "write"
	atomActionDelete    = "delete"
	atomActionManage    = "manage"
	atomActionPublish   = "publish"
	atomActionSubscribe = "subscribe"
	atomActionExecute   = "execute"
	atomActionList      = "list"

	atomActionAlarmRead        = "alarm_read"
	atomActionAlarmUpdate      = "alarm_update"
	atomActionAlarmDelete      = "alarm_delete"
	atomActionAlarmAssign      = "alarm_assign"
	atomActionAlarmAcknowledge = "alarm_acknowledge"
	atomActionAlarmResolve     = "alarm_resolve"
)

const (
	atomStatusActive    = "active"
	atomStatusInactive  = "inactive"
	atomStatusEnabled   = "enabled"
	atomStatusDisabled  = "disabled"
	atomStatusFrozen    = "frozen"
	atomStatusSuspended = "suspended"
	atomStatusDeleted   = "deleted"
)

const (
	atomKindDevice = "device"
	atomKindGroup  = "group"
	atomKindHuman  = "human"
)

const (
	atomObjectKindEntity   = "entity"
	atomObjectKindGroup    = "group"
	atomObjectKindResource = "resource"
	atomObjectKindTenant   = "tenant"
)

const atomScopeModeObject = "object"

const (
	atomScopeModeGroupDirectObjects     = "group_direct_objects"
	atomScopeModeGroupDescendantObjects = "group_descendant_objects"
)

const (
	atomObjectTypeResourceChannel = "resource:channel"
	atomObjectTypeResourceRule    = "resource:rule"
	atomObjectTypeResourceReport  = "resource:report"
	atomObjectTypeEntityDevice    = atomObjectKindEntity + ":" + atomKindDevice
)

const atomDecisionAllow = "allow"

const (
	atomInputKeyAction           = "action"
	atomInputKeyCredentialID     = "credentialId"
	atomInputKeyEntityID         = "entityId"
	atomInputKeyExternalID       = "externalId"
	atomInputKeyInput            = "input"
	atomInputKeyKind             = "kind"
	atomInputKeyName             = "name"
	atomInputKeyObjectGroupID    = "objectGroupId"
	atomInputKeyObjectKind       = "objectKind"
	atomInputKeyParentGroupID    = "parentGroupId"
	atomInputKeyParentID         = "parentId"
	atomInputKeySubjectID        = "subjectId"
	atomInputKeyProfileID        = "profileId"
	atomInputKeyProfileVersionID = "profileVersionId"
)

// Device type status. A deprecated or disabled device type accepts no new
// bindings; devices already bound to it are unaffected.
const (
	DeviceTypeStatusActive     = "active"
	DeviceTypeStatusDeprecated = "deprecated"
	DeviceTypeStatusDisabled   = "disabled"
)

// Device type version status. Only an active version can be bound to.
const (
	DeviceTypeVersionStatusDraft      = "draft"
	DeviceTypeVersionStatusActive     = "active"
	DeviceTypeVersionStatusDeprecated = "deprecated"
	DeviceTypeVersionStatusDisabled   = "disabled"
)

const (
	atomContextDomainID         = "domain_id"
	atomContextLegacyObjectType = "legacy_object_type"
)

const (
	atomAttributeCreatedAt = "created_at"
	atomAttributeGateways  = "gateways"
	// atomAttributeIsGateway marks a device as a gateway (spec §8 A12) — a
	// gateway is not a separate entity kind, just a device with this set.
	atomAttributeIsGateway = "is_gateway"
	atomAttributeMetadata  = "metadata"
	atomAttributeRoute     = "route"
	atomAttributeSource    = "source"
	atomAttributeStatus    = "status"
	atomAttributeTags      = "tags"
	atomAttributeUpdatedAt = "updated_at"
	atomAttributeUpdatedBy = "updated_by"
)

const atomAttributeSourceMagistrala = "magistrala"

const (
	atomGraphQLPath        = "/graphql"
	atomAuthIntrospectPath = "/auth/introspect"
)
