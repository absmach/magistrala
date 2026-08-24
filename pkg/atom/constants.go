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

// Object types are "<objectKind>:<resourceKind>" and must be derived from the
// Kind constants rather than spelled out: a literal that does not match the
// projected Kind makes every capability registered for it inapplicable, so
// each authorization check silently denies.
const (
	atomObjectTypeResourceChannel          = atomObjectKindResource + ":" + KindChannel
	atomObjectTypeResourceRule             = atomObjectKindResource + ":" + KindRule
	atomObjectTypeResourceReport           = atomObjectKindResource + ":" + KindReport
	atomObjectTypeResourceBootstrapConfig  = atomObjectKindResource + ":" + KindBootstrapConfig
	atomObjectTypeResourceBootstrapProfile = atomObjectKindResource + ":" + KindBootstrapProfile
	atomObjectTypeEntityDevice             = atomObjectKindEntity + ":" + atomKindDevice
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
	atomContextWorkspaceID      = "workspace_id"
	atomContextLegacyObjectType = "legacy_object_type"
)

const (
	atomAttributeCreatedAt = "created_at"
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
