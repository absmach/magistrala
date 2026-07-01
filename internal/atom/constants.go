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
	atomObjectTypeResourceChannel = "resource:channel"
	atomObjectTypeResourceRule    = "resource:rule"
	atomObjectTypeResourceReport  = "resource:report"
	atomObjectTypeResourceAlarm   = "resource:alarm"
)

const atomDecisionAllow = "allow"

const (
	atomInputKeyAction       = "action"
	atomInputKeyCredentialID = "credentialId"
	atomInputKeyEntityID     = "entityId"
	atomInputKeyInput        = "input"
	atomInputKeyKind         = "kind"
	atomInputKeyName         = "name"
	atomInputKeyObjectKind   = "objectKind"
	atomInputKeySubjectID    = "subjectId"
)

const (
	atomContextDomainID         = "domain_id"
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

const atomServiceTokenJournal = "journal"

const (
	atomGraphQLPath        = "/graphql"
	atomAuthIntrospectPath = "/auth/introspect"
)
